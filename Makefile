.PHONY: docs

build:
	go build .
	mv ./edg ~/dev/bin

test:
	go test ./... -v --cover

bench:
	go test ./... -bench=. -benchmem -count=1 -benchtime=100ms

integration_test_crdb:
	URL="postgres://root@localhost:26257?sslmode=disable" \
	DRIVER="pgx" \
	go test ./pkg -v -db -rng-seed 42

docs:
	(cd docs && hugo server --disableFastRender)

teardown:
	- docker compose -f _examples/compose_crdb.yml down
	- docker rm cockroachdb -f
	- docker rm oracle -f
	- rm ./tpc-c