package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/codingconcepts/db-chaos/pkg/repo"
	"github.com/codingconcepts/db-chaos/pkg/runner"
	"github.com/samber/lo"
)

func main() {
	log.SetFlags(0)

	var r runner.WorkloadRunner

	database := flag.String("database", "postgres", "the database under test [oracle | postgres]")
	url := flag.String("url", "", "database connection string")
	ns := flag.String("namespace", "default", "database namespace")
	chaosNS := flag.String("chaos-namespace", "chaos-mesh", "chaos mesh namespace")
	expDuration := flag.Duration("experiment-duration", time.Second*30, "length of each chaos experiment")
	readyTimeout := flag.Duration("ready-timeout", time.Second*60, "amount of time to wait for ready pods")
	flag.BoolVar(&r.Reseed, "reseed", false, "reseed the database with test data")
	flag.IntVar(&r.Accounts, "accounts", 10000, "number of accounts in bank")
	flag.IntVar(&r.Active, "active", 1000, "number of active accounts in bank")
	flag.Float64Var(&r.InitialBalance, "balance", 10000, "initial account balances")
	flag.Parse()

	repo, err := selectRepo(*database, *url)
	if err != nil {
		log.Fatalf("error selecting repo: %v", err)
	}

	notify := make(chan string, 1)

	chaosRunner, err := runner.NewChaosRunner(repo, *ns, *chaosNS, *expDuration, *readyTimeout, notify)
	if err != nil {
		log.Fatalf("error creating chaos runner: %v", err)
	}

	// Run chaos runner on another thread so we don't block the workload.
	go func() {
		time.Sleep(time.Second * 10)
		if err = chaosRunner.Run(); err != nil {
			log.Fatalf("error running chaos experiments: %v", err)
		}
		close(notify)
	}()

	results, err := r.Run(repo, notify)
	if err != nil {
		log.Fatalf("error running simulation: %v", err)
	}

	log.Printf("Total")
	log.Printf("\terrors:   %d", results.TotalErrors)
	log.Printf("\tdowntime: %s", results.TotalDowntime)

	keys := lo.Keys(results.Stats)
	sort.Strings(keys)
	for _, key := range keys {
		stats := results.Stats[key]
		log.Printf("\n%s", key)
		log.Printf("\terrors:   %d", stats.ErrorCount)
		log.Printf("\tdowntime: %s", stats.Downtime)
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
