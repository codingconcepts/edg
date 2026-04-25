# MongoDB

A basic MongoDB example demonstrating collection creation, document inserts with generated UUIDs and emails, and cross-collection references.

## Setup

```sh
docker compose -f cmd/harness/compose/compose_mongo.yml up -d
```

MongoDB takes a few seconds to become ready:

```sh
until docker exec mongo mongosh --quiet --eval "db.runCommand({ping:1})" > /dev/null 2>&1; do
  echo "waiting for mongo..."; sleep 2
done && echo "mongo ready"
```

## Run

```sh
go run ./cmd/edg up \
--driver mongodb \
--config _examples/mongodb/mongodb.yaml \
--url "mongodb://localhost:27017/edg"

go run ./cmd/edg seed \
--driver mongodb \
--config _examples/mongodb/mongodb.yaml \
--url "mongodb://localhost:27017/edg"
```

Check data:

```sh
docker exec mongo mongosh edg --quiet --eval "db.customer.countDocuments(); db.account.countDocuments();"
docker exec mongo mongosh edg --quiet --eval "db.customer.find().limit(5).toArray();"
docker exec mongo mongosh edg --quiet --eval "db.account.find().limit(5).toArray();"
```

```sh
go run ./cmd/edg deseed \
--driver mongodb \
--config _examples/mongodb/mongodb.yaml \
--url "mongodb://localhost:27017/edg"

go run ./cmd/edg down \
--driver mongodb \
--config _examples/mongodb/mongodb.yaml \
--url "mongodb://localhost:27017/edg"
```
