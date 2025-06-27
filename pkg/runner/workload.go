package runner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/codingconcepts/db-chaos/pkg/repo"
	"github.com/fatih/color"
	"github.com/samber/lo"
)

var (
	pink = color.RGB(236, 63, 150).SprintFunc()
	blue = color.RGB(0, 252, 237).SprintFunc()
)

type WorkloadRunner struct {
	Reseed         bool
	Accounts       int
	Active         int
	InitialBalance float64
}

type ExperimentStats struct {
	ErrorCount int
	Downtime   time.Duration
}

type Results struct {
	TotalErrors   int
	TotalDowntime time.Duration

	Stats map[string]ExperimentStats
}

func (r *WorkloadRunner) Run(repo repo.Repo, notify <-chan string) (Results, error) {
	if r.Reseed {
		if err := repo.Deinit(); err != nil {
			return Results{}, fmt.Errorf("error running deinit: %w", err)
		}
		log.Println("ran deinit successfully")

		if err := repo.Init(r.Accounts, r.InitialBalance); err != nil {
			return Results{}, fmt.Errorf("error initialising database: %w", err)
		}
		log.Println("ran init successfully")
	}

	accountIDs, err := repo.FetchIDs(r.Accounts)
	if err != nil {
		return Results{}, fmt.Errorf("error fetching ids ahead of test: %w", err)
	}

	var errorCount int
	var totalDowntime time.Duration

	var currentExperiment string
	experimentStats := map[string]ExperimentStats{}

	// Perform a transfer every 100ms.
	transfers := time.Tick(time.Millisecond * 100)
	for {
		select {
		case exp, ok := <-notify:
			if !ok {
				return Results{
					TotalErrors:   errorCount,
					TotalDowntime: totalDowntime,
					Stats:         experimentStats,
				}, nil
			}

			currentExperiment = exp

		case <-transfers:
			ids := lo.Samples(accountIDs, 2)

			taken, err := repo.PerformTransfer(ids[0], ids[1], rand.Float64()*100)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("error: %v", err)
				errorCount++
				totalDowntime += taken

				increment(experimentStats, currentExperiment, taken)
			}

			latencyMS := fmt.Sprintf("%dms", taken.Milliseconds())
			totalDowntimeS := fmt.Sprintf("%0.2fs", totalDowntime.Seconds())

			fmt.Printf(
				"latency: %s, errors: %s, total downtime: %s\n",
				blue(latencyMS),
				pink(errorCount),
				pink(totalDowntimeS),
			)
		}
	}
}

func increment(m map[string]ExperimentStats, name string, d time.Duration) {
	stats, ok := m[name]
	if !ok {
		stats = ExperimentStats{}
	}

	stats.ErrorCount++
	stats.Downtime += d

	m[name] = stats
}
