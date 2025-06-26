package repo

import (
	"context"
	"fmt"
	"log"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(url string) (*PostgresRepo, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing connection string: %w", err)
	}
	cfg.MaxConns = 3

	// These two values combined are what drive the server.shutdown.connections.timeout
	// setting in CockroachDB.
	cfg.MaxConnLifetime = time.Second * 15
	cfg.MaxConnLifetimeJitter = time.Second * 5

	db, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("error testing database connection: %w", err)
	}

	return &PostgresRepo{
		db: db,
	}, nil
}

func (p *PostgresRepo) Init(rowCount int, balance float64) error {
	const tableStmt = `CREATE TABLE account (
										   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
											 balance DECIMAL NOT NULL
										 )`

	if _, err := p.db.Exec(context.Background(), tableStmt); err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	const insertStmt = `INSERT INTO account (balance)
											SELECT $2
											FROM generate_series(1, $1)`

	if _, err := p.db.Exec(context.Background(), insertStmt, rowCount, balance); err != nil {
		return fmt.Errorf("seeding table: %w", err)
	}

	log.Println("created and seeded table successfully")
	return nil
}

func (p *PostgresRepo) Deinit() error {
	const stmt = `DROP TABLE account`

	_, err := p.db.Exec(context.Background(), stmt)
	if err != nil {
		return fmt.Errorf("dropping table: %w", err)
	}

	log.Println("dropped table successfully")
	return nil
}

func (p *PostgresRepo) FetchIDs(count int) ([]any, error) {
	const stmt = `SELECT id FROM account ORDER BY random() LIMIT $1`

	rows, err := p.db.Query(context.Background(), stmt, count)
	if err != nil {
		return nil, fmt.Errorf("querying for rows: %w", err)
	}

	var accountIDs []any
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning id: %w", err)
		}
		accountIDs = append(accountIDs, id)
	}

	return accountIDs, nil
}

func (p *PostgresRepo) PerformTransfer(from, to any, amount float64) (elapsed time.Duration, err error) {
	// Timeout queries after 5s (configure to your requirements).
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	start := time.Now()
	defer func() {
		elapsed = time.Since(start)
	}()

	const stmt = `UPDATE account
									SET balance = CASE 
										WHEN id = $1 THEN balance - $3
										WHEN id = $2 THEN balance + $3
									END
								WHERE id IN ($1, $2);`

	// Wrapping pgx query with crdbpgx to ensure retryable requests are retried
	// for both databases.
	txOptions := pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}
	err = crdbpgx.ExecuteTx(timeout, p.db, txOptions, func(tx pgx.Tx) error {
		_, err := p.db.Exec(timeout, stmt, from, to, amount)
		return err
	})

	return
}

func (p *PostgresRepo) IsReady() (bool, error) {
	const stmt = `SELECT COUNT(*) AS underreplicated_ranges
								FROM crdb_internal.ranges
								WHERE array_length(replicas, 1) < 3`

	row := p.db.QueryRow(context.Background(), stmt)

	var underreplicatedRanges int
	if err := row.Scan(&underreplicatedRanges); err != nil {
		return false, fmt.Errorf("checking ready: %w", err)
	}

	return true, nil
}
