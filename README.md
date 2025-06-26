# db-chaos
A simple wrapper around Chaos Mesh that tests database resilience in Kubernetes.

### Usage

Help text

```sh
dbchaos --help
Usage of dbchaos:
  -accounts int
        number of accounts in bank (default 10000)
  -active int
        number of active accounts in bank (default 1000)
  -balance float
        initial account balances (default 10000)
  -database string
        the database under test [oracle | postgres] (default "postgres")
  -reseed
        reseed the database with test data
  -url string
        database connection string
```

### Supported databases

CockroachDB

```sh
go run main.go \
--database postgres \
--url "postgres://root@localhost:26257?sslmode=disable" \
--reseed
```

Postgres



MySQL



Oracle

