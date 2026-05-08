# HiGoOS Database

This folder contains the PostgreSQL 16 schema and development seed for the Go backend.

## Requirements

- PostgreSQL 16
- `pgcrypto`, available in a normal PostgreSQL install
- `pgvector`, required by `semantic_embeddings.embedding vector(1536)`

On macOS with Homebrew:

```sh
brew install postgresql@16 pgvector
brew services start postgresql@16
```

If `psql` cannot find PostgreSQL 16, add it for the current shell:

```sh
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"
```

## Create A Local Database

```sh
createdb higoos_dev
psql -d higoos_dev -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;'
psql -d higoos_dev -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

If the `vector` extension is not installed, `CREATE EXTENSION IF NOT EXISTS vector` and the migration will fail. Install `pgvector` first, or run static SQL checks only until the extension is available. The initial schema intentionally keeps the real `vector(1536)` column so local development matches Linux deployment.

## Migrate And Seed

From the repository root:

```sh
psql -v ON_ERROR_STOP=1 -d higoos_dev -f server-go/db/migrations/0001_core.sql
psql -v ON_ERROR_STOP=1 -d higoos_dev -f server-go/db/seeds/dev_seed.sql
```

The seed mirrors `web-pc/src/data/higoos.ts` for dock apps, desktop windows, spaces, demo files, storage pools, disks, Agent templates, Assistant messages, metrics, and alerts.

## Reset

```sh
dropdb --if-exists higoos_dev
createdb higoos_dev
psql -v ON_ERROR_STOP=1 -d higoos_dev -f server-go/db/migrations/0001_core.sql
psql -v ON_ERROR_STOP=1 -d higoos_dev -f server-go/db/seeds/dev_seed.sql
```

## Fixture Files

Development fixture content lives under `server-go/fixtures/nas-root`. Backend dev adapters can mount or copy this tree to simulate NAS spaces without touching real user data.

## Linux Deployment Notes

- Install PostgreSQL 16 and the matching `pgvector` package for the distribution before running migrations.
- Run migrations with a database user that can create extensions, or create `pgcrypto` and `vector` once with an administrator role before application startup.
- Linux-only adapters for disks, SMART, Docker, system logs, tunnels, downloads, and media workers should write runtime facts into these tables; Mac development can keep using devstub adapters and fixture files.
- Keep seed data out of production. Use `dev_seed.sql` only for local development, demos, and integration tests that explicitly reset their database.
