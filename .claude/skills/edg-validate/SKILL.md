---
name: edg-validate
description: Validate an edg YAML config file, interpret errors, and suggest fixes.
user-invocable: true
---

# edg Config Validator

You validate edg YAML workload configurations by running `edg validate` and interpreting the results.

## Steps

1. **Identify the config file.** If the user specifies a path, use it. Otherwise, look for a YAML file in the current directory or `_examples/` that looks like an edg config.

2. **Identify the driver.** If the user specifies a driver, use it. Otherwise, infer from the config content:
   - `STRING` type, `gen_random_uuid()`, `string_to_array` → `pgx`
   - `CHAR(36)`, `UUID()`, `JSON_TABLE` → `mysql`
   - `UNIQUEIDENTIFIER`, `NEWID()`, `OPENJSON`, `NVARCHAR` → `mssql`
   - `VARCHAR2`, `SYSTIMESTAMP`, `XMLTABLE`, `CONNECT BY` → `oracle`
   - If unclear, default to `pgx`

3. **Run validation:**
   ```sh
   edg validate --driver <driver> --config <path>
   ```

4. **Interpret the output.** If validation fails, explain what went wrong and suggest a fix. Common issues include:
   - **Missing section**: A required section is missing from the config
   - **Invalid expression**: A query arg uses an unknown function or has a syntax error
   - **Type mismatch**: An expression returns the wrong type for its context
   - **Missing dataset**: A `ref_*` call references a dataset that no `init` or `seed` query populates
   - **Invalid query type**: Using `query` when `exec` is needed, or vice versa
   - **Batch config errors**: Missing `count`/`size` for batch types, or using batch args with non-batch types
   - **Transaction constraint violations**: Using `exec_batch`/`query_batch` or `prepared: true` inside a transaction, or an empty transaction with no queries

5. **Apply fixes.** If the user asks, edit the config file to fix the issues and re-validate.

## Common Fixes

| Error pattern | Likely cause | Fix |
|---|---|---|
| Unknown function `foo` | Typo or unsupported expression | Check spelling against the expressions reference |
| Dataset `x` not found | Missing `init` query or name mismatch | Add an `init` query named `x` with `type: query`, or fix the dataset name |
| Expected exec, got query | SELECT used with `type: exec` | Change to `type: query` |
| Batch requires count | `exec_batch` without `count` field | Add `count` and `size` fields |
| Expression compile error | Invalid expr-lang syntax | Check for missing quotes, unmatched parens, or invalid operators |
| Cannot be a batch type inside a transaction | `exec_batch`/`query_batch` in transaction | Change to `exec`/`query` and move batching outside the transaction |
| Cannot use prepared statements inside a transaction | `prepared: true` in transaction | Remove `prepared: true` from queries inside the transaction |
| Must contain at least one query | Empty transaction | Add queries to the transaction or remove it |
