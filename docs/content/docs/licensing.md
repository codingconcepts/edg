---
title: Licensing
weight: 9
---

# Licensing

edg is free to use for PostgreSQL/CockroachDB and MySQL workloads. Enterprise drivers require a paid license.

## Free vs Enterprise Drivers

| Driver | Flag Value | License Required |
|---|---|---|
| PostgreSQL / CockroachDB | `pgx` | No |
| MySQL | `mysql` | No |
| Oracle | `oracle` | Yes |
| Microsoft SQL Server | `mssql` | <strong>Yes</strong> |
| AWS Aurora DSQL | `dsql` | <strong>Yes</strong> |
| Google Cloud Spanner | `spanner` | <strong>Yes</strong> |

Commands that don't connect to a database (`repl`, `validate config`, and bare expression evaluation) work without a license regardless of driver.

## Obtaining a License

Contact [lic@edg.run](mailto:lic@edg.run) to purchase a license key. A license includes:

- **Licensed drivers** - each license covers one or more enterprise drivers (e.g. Oracle + MSSQL)
- **Expiry date** - licenses are time-limited and must be renewed before they expire

## Using a License

Provide your license key via the `--license` flag or the `EDG_LICENSE` environment variable:

```sh
# Flag.
edg all \
  --driver oracle \
  --license "your-license-key" \
  --config workload.yaml \
  --url "oracle://user:pass@localhost:1521/XEPDB1"

# Environment variable.
export EDG_LICENSE="your-license-key"

edg all \
  --driver oracle \
  --config workload.yaml \
  --url "oracle://user:pass@localhost:1521/XEPDB1"
```

The environment variable is checked when the `--license` flag is not set, so you can export it once and run multiple commands without repeating it.

## License Validation

When you use an enterprise driver, edg checks the license before connecting to the database. If validation fails you'll see one of these errors:

| Error | Meaning |
|---|---|
| `driver "X" requires a license` | No license was provided. Set `--license` or `EDG_LICENSE`. |
| `license expired on YYYY-MM-DD` | License has passed its expiry date. Contact [lic@edg.run](mailto:lic@edg.run) to renew. |
| `license does not include driver "X"` | License is valid but doesn't cover this driver. |

## Checking Your License

Use `validate license` to verify a license key and inspect its details without connecting to a database:

```sh
edg validate license --driver oracle --license "your-license-key"
```

```
License info:
  ID:         acme-corp
  Email:      admin@acme.com
  Drivers:    [oracle mssql]
  Issued at:  2025-01-15
  Expires at: 2026-01-15
License is valid for driver "oracle".
```

If you pass a free driver like `pgx` or `mysql`, the output confirms no license is needed:

```sh
edg validate license --driver pgx --license "your-license-key"
```

```
License info:
  ID:         acme-corp
  Email:      admin@acme.com
  Drivers:    [oracle mssql]
  Issued at:  2025-01-15
  Expires at: 2026-01-15
Driver "pgx" does not require a license.
```

This prints the license holder, licensed drivers, issue/expiry dates, and whether the license covers the requested `--driver`. Useful for troubleshooting license issues or confirming a renewal was applied.

## How It Works

Licenses are cryptographically signed using Ed25519. The public key is embedded in the edg binary, so verification is offline (no network calls are made).
