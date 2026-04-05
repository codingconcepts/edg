.PHONY: docs

build:
	go build .
	mv ./edg ~/dev/bin

test:
	go test ./... -v --cover

bench:
	go test ./... -bench=. -benchmem -count=1 -benchtime=100ms

integration_test:
	CGO_ENABLED=0 GOOS=linux go test ./pkg -v -c -o edg-test
	docker build -t edg-test -f Dockerfile.test .
	docker compose -f docker-compose.test.yml up --abort-on-container-exit --force-recreate
	docker compose -f docker-compose.test.yml down
	rm -f edg-test

docs:
	(cd docs && bundle install)
	(cd docs && bundle exec jekyll serve)

teardown:
	- docker compose -f _examples/compose_crdb.yml down
	- docker rm cockroachdb -f
	- docker rm oracle -f
	- rm ./tpc-c