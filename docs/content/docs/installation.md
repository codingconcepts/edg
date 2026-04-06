---
title: Installation
weight: 1
---

# Installation

```sh
go install github.com/codingconcepts/edg@latest
```

Or pull the Docker image:

```sh
docker pull codingconcepts/edg:v0.1.0
```

Images are published for both `linux/amd64` and `linux/arm64`. Pass flags and mount your config file:

```sh
docker run --rm \
  -v $(pwd)/_examples/tpcc:/config \
  codingconcepts/edg:v0.1.0 all \
  --driver pgx \
  --config /config/crdb.yaml \
  --url "postgres://root@host.docker.internal:26257?sslmode=disable"
```

Or build from source:

```sh
git clone https://github.com/codingconcepts/edg
cd edg
go build -o edg .
```
