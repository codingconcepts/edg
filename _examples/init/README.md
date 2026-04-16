# Init

Generates a config file based on an existing database.

## CockroachDB

### Setup

```sh
docker compose -f _examples/compose_crdb.yml up -d

go run ./cmd/edg up \
--driver pgx \
--config _examples/normal/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Init

```sh
go run ./cmd/edg init \
--driver pgx \
--url "postgres://root@localhost:26257?sslmode=disable" \
--schema public > _examples/init/crdb.yaml

go run ./cmd/edg seed \
--driver pgx \
--config _examples/init/crdb.yaml \
--url "postgres://root@localhost:26257?sslmode=disable"
```

### Check

```sql
SELECT 'users', count(*) FROM users
UNION ALL SELECT 'vehicles', count(*) FROM vehicles
UNION ALL SELECT 'rides', count(*) FROM rides;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver pgx \
--config _examples/init/crdb.yaml \
--url "postgres://root@localhost:26257/movr?sslmode=disable"

go run ./cmd/edg down \
--driver pgx \
--config _examples/init/crdb.yaml \
--url "postgres://root@localhost:26257/movr?sslmode=disable"
```

## MySQL

### Setup

```sh
docker compose -f _examples/compose_mysql.yml up -d

go run ./cmd/edg up \
--driver mysql \
--config _examples/normal/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

### Init

```sh
go run ./cmd/edg init \
--driver mysql \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true" \
--database defaultdb > _examples/init/mysql.yaml
```

### Seed

```sh
go run ./cmd/edg seed \
--driver mysql \
--config _examples/init/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

### Check

```sql
SELECT 'product', count(*) FROM product
UNION ALL SELECT 'review', count(*) FROM review;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver mysql \
--config _examples/init/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"

go run ./cmd/edg down \
--driver mysql \
--config _examples/init/mysql.yaml \
--url "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"
```

## MSSQL

### Setup

```sh
docker compose -f _examples/compose_mssql.yml up -d

go run ./cmd/edg up \
--driver mssql \
--config _examples/normal/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable"
```

### Init

```sh
go run ./cmd/edg init \
--driver mssql \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable" \
--schema dbo > _examples/init/mssql.yaml
```

### Seed

```sh
go run ./cmd/edg seed \
--driver mssql \
--config _examples/init/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable"
```

### Check

```sql
SELECT 'product', count(*) FROM product
UNION ALL SELECT 'review', count(*) FROM review;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver mssql \
--config _examples/init/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable"

go run ./cmd/edg down \
--driver mssql \
--config _examples/init/mssql.yaml \
--url "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable"
```

## Oracle

### Setup

```sh
docker compose -f _examples/compose_oracle.yml up -d

go run ./cmd/edg up \
--driver oracle \
--config _examples/normal/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

### Init

```sh
go run ./cmd/edg init \
--driver oracle \
--url "oracle://system:password@localhost:1521/defaultdb" \
--schema SYSTEM > _examples/init/oracle.yaml
```

### Seed

```sh
go run ./cmd/edg seed \
--driver oracle \
--config _examples/init/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```

### Check

```sql
SELECT 'product', count(*) FROM product
UNION ALL SELECT 'review', count(*) FROM review;
```

### Teardown

```sh
go run ./cmd/edg deseed \
--driver oracle \
--config _examples/init/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"

go run ./cmd/edg down \
--driver oracle \
--config _examples/init/oracle.yaml \
--url "oracle://system:password@localhost:1521/defaultdb"
```
