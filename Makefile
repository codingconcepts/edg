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

sqlserver:
	docker run \
		-d \
		--name sqlserver \
		-e ACCEPT_EULA=Y \
		-e 'MSSQL_SA_PASSWORD=P4ssw0rd' \
		-p 1433:1433 \
		mcr.microsoft.com/azure-sql-edge:latest

integration_test_sqlserver:
	URL="sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
	DRIVER="sqlserver" \
	go test ./pkg -v -db -rng-seed 42
	
docs:
	(cd docs && hugo server --disableFastRender)

teardown:
	- docker ps -aq | xargs docker rm -f
	- rm ./tpc-c