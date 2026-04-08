# Config Includes

This example demonstrates the `!include` directive for reusing reference data and expressions across workload files.

## Usage

Use the `!include` YAML tag to pull in content from another file:

```yaml
globals: !include shared/globals.yaml
up: !include shared/schema.yaml
down: !include shared/teardown.yaml
run: !include shared/run_queries.yaml
```

Paths are resolved relative to the file containing the `!include` directive.

## Structure

```
includes/
  crdb.yaml                   # Main workload file
  shared/
    globals.yaml              # Shared global variables
    schema.yaml               # Shared schema (up) queries
    teardown.yaml             # Shared teardown (down) queries
    run_queries.yaml          # Shared benchmark queries
```

## What can be included

An `!include` can appear anywhere a YAML value is expected:

- **Mapping value** — replace a key's value with the content of a file:
  ```yaml
  globals: !include shared/globals.yaml
  ```

- **Sequence value** — replace an entire list:
  ```yaml
  up: !include shared/schema.yaml
  ```

- **Sequence item** — splice items from an included file into a list:
  ```yaml
  run:
    - name: local_query
      query: SELECT 1
    - !include shared/extra_queries.yaml
  ```

Nested includes are supported (an included file can itself use `!include`). Circular includes are detected and produce an error.
