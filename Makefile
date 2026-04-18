.PHONY: docs

validate_version:
ifndef VERSION
	$(error VERSION is undefined)
endif

docker_push: validate_version
	@docker build --platform linux/amd64 \
		--build-arg VERSION=${VERSION} \
		-t codingconcepts/edg:linux_amd64_${VERSION} \
		--push \
		.

	@docker build --platform linux/arm64 \
		--build-arg VERSION=${VERSION} \
		-t codingconcepts/edg:linux_arm64_${VERSION} \
		--push \
		.

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

integration_test_schema_crdb:
	URL="postgres://root@localhost:26257?sslmode=disable" \
	DRIVER="pgx" \
	go test ./pkg/schema -v -db

integration_test_schema_mysql:
	URL="root:password@tcp(localhost:3306)/defaultdb" \
	DRIVER="mysql" \
	go test ./pkg/schema -v -db

integration_test_schema_mssql:
	URL="sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
	DRIVER="mssql" \
	go test ./pkg/schema -v -db

integration_test_schema_oracle:
	URL="oracle://system:password@localhost:1521/defaultdb" \
	DRIVER="oracle" \
	go test ./pkg/schema -v -db

harness_crdb:
	go run ./cmd/harness -db crdb

harness_mysql:
	go run ./cmd/harness -db mysql

harness_mssql:
	go run ./cmd/harness -db mssql

harness_oracle:
	go run ./cmd/harness -db oracle

harness_all: harness_crdb harness_mysql harness_mssql harness_oracle
	echo "done"

docs:
	(cd docs && hugo server --disableFastRender --openBrowser)

teardown:
	- docker ps -aq | xargs docker rm -f
	- rm ./edg ./harness