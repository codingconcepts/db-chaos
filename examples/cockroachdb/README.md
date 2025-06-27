Setup k3d cluster

```sh
k3d cluster create local \
--api-port 6550 \
-p "26257:26257@loadbalancer" \
-p "8080:8080@loadbalancer" \
--agents 2
```

Setup CockroachDB

```sh
kubectl apply -f examples/cockroachdb/v25.2.1.yaml
kubectl wait --for=jsonpath='{.status.phase}'=Running pods --all -n crdb --timeout=300s
kubectl exec -it -n crdb cockroachdb-0 -- /cockroach/cockroach init --insecure
```

Setup Chaos Mesh

```sh
curl -sSL https://mirrors.chaos-mesh.org/v2.7.0/install.sh | bash -s -- -r containerd

# OR

curl -sSL https://mirrors.chaos-mesh.org/v2.7.2/install.sh | bash -s -- --k3s

```

Run workload

```sh
go run main.go \
--database postgres \
--url "postgres://root@localhost:26257?sslmode=disable" \
--reseed \
--namespace crdb \
--chaos-namespace chaos-mesh \
--experiment-duration 30s \
--ready-timeout 60s
```

Monitor workload

```sh
see -n 3 cockroach sql --insecure --execute "
  SELECT
  SUM((metrics->>'ranges.underreplicated')::DECIMAL) AS total_underreplicated_ranges
  FROM crdb_internal.kv_store_status
  LIMIT 1"
```