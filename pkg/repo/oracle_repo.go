package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

type OracleRepo struct {
	db *sql.DB
}

func NewOracleRepo(url string) (*OracleRepo, error) {
	db, err := sql.Open("oracle", url)
	if err != nil {
		return nil, fmt.Errorf("opening databse connection: %w", err)
	}

	if err = db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("error testing database connection: %w", err)
	}

	return &OracleRepo{
		db: db,
	}, nil
}

func (p *OracleRepo) Init(rowCount int, balance float64) error {
	return nil
}

func (p *OracleRepo) Deinit() error {
	return nil
}

func (p *OracleRepo) FetchIDs(count int) ([]any, error) {
	var ids []any
	return ids, nil
}

func (p *OracleRepo) PerformTransfer(from, to any, amount float64) (time.Duration, error) {
	return 0, nil
}

func (p *OracleRepo) IsReady() (bool, error) {
	return true, nil
}
