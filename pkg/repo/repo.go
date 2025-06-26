package repo

import "time"

type Repo interface {
	Init(rowCount int, balance float64) error
	Deinit() error
	FetchIDs(count int) ([]any, error)
	PerformTransfer(from, to any, amount float64) (time.Duration, error)
	IsReady() (bool, error)
}
