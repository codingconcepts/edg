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

mssql:
	docker run \
		-d \
		--name mssql \
		-e ACCEPT_EULA=Y \
		-e 'MSSQL_SA_PASSWORD=P4ssw0rd' \
		-p 1433:1433 \
		mcr.microsoft.com/azure-sql-edge:latest

integration_test_mssql:
	URL="sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
	DRIVER="mssql" \
	go test ./pkg -v -db -rng-seed 42

harness_crdb:
	go run ./cmd/harness -db crdb

harness_mysql:
	go run ./cmd/harness -db mysql

harness_mssql:
	go run ./cmd/harness -db mssql

harness_oracle:
	go run ./cmd/harness -db oracle

harness_all: harness_crdb harness_mysql harness_mssql
	echo "done"

docs:
	(cd docs && hugo server --disableFastRender)

teardown:
	- docker ps -aq | xargs docker rm -f
	- rm ./tpc-c