test:
	go test ./... -v --cover

bench:
	go test ./... -bench=. -benchmem -count=1 -benchtime=100ms

teardown:
	- docker compose -f _examples/compose_crdb.yml down
	- docker rm oracle -f
	- rm ./tpc-c