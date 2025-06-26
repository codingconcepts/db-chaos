package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/codingconcepts/db-chaos/pkg/repo"
	"github.com/fatih/color"
	"github.com/samber/lo"
)

var (
	pink = color.RGB(236, 63, 150).SprintFunc()
	blue = color.RGB(0, 252, 237).SprintFunc()
)

func main() {
	log.SetFlags(0)

	var r runner

	database := flag.String("database", "postgres", "the database under test [oracle | postgres]")
	url := flag.String("url", "", "database connection string")
	flag.BoolVar(&r.reseed, "reseed", false, "reseed the database with test data")
	flag.IntVar(&r.accounts, "accounts", 10000, "number of accounts in bank")
	flag.IntVar(&r.active, "active", 1000, "number of active accounts in bank")
	flag.Float64Var(&r.initialBalance, "balance", 10000, "initial account balances")
	flag.Parse()

	repo, err := selectRepo(*database, *url)
	if err != nil {
		log.Fatalf("error selecting repo: %v", err)
	}

	if err := r.run(repo); err != nil {
		log.Fatalf("error running simulation: %v", err)
	}
}

func selectRepo(database, url string) (repo.Repo, error) {
	switch strings.ToLower(database) {
	case "oracle":
		return repo.NewOracleRepo(url)

	case "postgres":
		return repo.NewPostgresRepo(url)

	default:
		return nil, fmt.Errorf("unsupported database: %q", database)
	}
}

type runner struct {
	reseed         bool
	accounts       int
	active         int
	initialBalance float64
}

func (r *runner) run(repo repo.Repo) error {
	if r.reseed {
		if err := repo.Deinit(); err != nil {
			log.Printf("error destroying database: %v", err)
		}

		if err := repo.Init(r.accounts, r.initialBalance); err != nil {
			log.Fatalf("error initialising database: %v", err)
		}
	}

	accountIDs, err := repo.FetchIDs(r.accounts)
	if err != nil {
		log.Fatalf("error fetching ids ahead of test: %v", err)
	}

	var errorCount int
	var totalDowntime time.Duration

	// Perform a transfer every 100ms.
	for range time.NewTicker(time.Millisecond * 100).C {
		ids := lo.Samples(accountIDs, 2)

		taken, err := repo.PerformTransfer(ids[0], ids[1], rand.Float64()*100)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("error: %v", err)
			errorCount++
			totalDowntime += taken
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

	panic("unexected app termination")
}
