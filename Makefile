test:
	go test ./... -v --cover

teardown:
	- docker compose -f _examples/compose_crdb.yml down
	- docker rm oracle -f
	- rm ./tpc-c