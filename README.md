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
  -chaos-namespace string
        chaos mesh namespace (default "chaos-mesh")
  -database string
        the database under test [oracle | postgres] (default "postgres")
  -experiment-duration duration
        length of each chaos experiment (default 30s)
  -namespace string
        database namespace (default "default")
  -ready-timeout duration
        amount of time to wait for ready pods (default 1m0s)
  -reseed
        reseed the database with test data
  -url string
        database connection string
```

### Supported databases

* CockroachDB - [example](examples/cockroachdb/README.md)
* Oracle

```sh
go run main.go \
--database oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
--reseed
```

```sql
SHOW RANGES FROM DATABASE defaultdb WITH TABLES;

SELECT *
FROM [SHOW RANGES FROM DATABASE defaultdb WITH TABLES] r
JOIN

SELECT
	range_id,
	ARRAY_LENGTH(replicas, 1) AS actual_replica_count
FROM
	crdb_internal.ranges
WHERE ARRAY_LENGTH(replicas, 1) < 3
ORDER BY actual_replica_count ASC;


SELECT
	SUM(metrics->'ranges.underreplicated') AS total_underreplicated_ranges
FROM crdb_internal.kv_store_status LIMIT 1;
```

  