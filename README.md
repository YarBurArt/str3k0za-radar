# str3k0za radar

This is a Telegram bot written in Go that works as an automated cybersecurity advisor. It builds a daily digest out of MITRE ATT&CK techniques and Common Weakness Enumerations, weighted towards the groups that show up in real DFIR reporting, and delivers it to the user on a schedule they pick.

Every user starts with the digest switched off. Enabling it from the chat assigns a delivery time (18:30 UTC unless the user already picked one with `/settime`), and the filter is narrowed down interactively: either by APT group or by the source country those groups are attributed to. The digest content itself is identical whether it is pulled on demand with `/digest` or pushed by the scheduler.

> Both binaries talk to Telegram API through a SOCKS5 proxy for some reason...

Public data from:

- https://github.com/mitre/cti
- https://cwe.mitre.org/data/downloads.html (research dataset)

## Requirements

- Go 1.26 or newer
- PostgreSQL 18
- PostgreSQL client tools (`psql`) for the app migrations
- `river` CLI for the job queue schema: 
    > `go install github.com/riverqueue/river/cmd/river@latest`
- The MITRE datasets dropped into `data/` (already committed, see `.gitignore` for the large sources)

## Installation

### from a release

Grab the two binaries and the `data/` directory, then keep them next to each other.

```bash
./bot
./worker
```

### from source

```bash
git clone https://github.com/YarBurArt/str3k0za-radar.git
```
```bash
cd ./str3k0za-radar
```
Build main bot
```bash
go build -o bin/bot ./cmd/bot
```
Build river worker
```bash
go build -o bin/worker ./cmd/worker
```

The binaries read the datasets from `./data` relative to the working directory, so run them from the repository root.

## Usage

### configuration

| var | what | why |
| --- | --- | --- |
| `BOT_TOKEN` | bot token from @BotFather | both binaries need it, the worker sends digests with it |
| `DB_URL` | Postgres DSN | preferences and the River job tables live in the same database |
| `SOCKS5_PROXY` | SOCKS5 endpoint | only to move it from the default port |
| `TEST_DATABASE_URL` | throwaway Postgres DSN | enables the repository tests, they skip without it |


### migrations

The application schema is plain SQL in `migrations/`, applied in filename order.

```bash
psql "$DB_URL" -f migrations/001_create_users.sql
psql "$DB_URL" -f migrations/002_create_preferences.sql
psql "$DB_URL" -f migrations/003_digest_disabled_by_default.sql
```

The job queue schema belongs to River and is applied by River itself, which tracks its own versions. Pls do not copy it into `migrations/`.

```bash
river migrate-up --database-url "$DB_URL"
```

Check what is pending without applying anything with `river migrate-up --database-url "$DB_URL" --dry-run`.

### run

```bash
./bin/bot
./bin/worker
```


### commands

|what| why |
| --- | --- |
| `/start`, `/help` | registers the user on first contact, shows the command list and the filter keyboard |
| `/enable` | switches the daily digest on, assigning the default delivery time if none is set |
| `/disable` | switches the daily digest off, keeping the chosen time for next time |
| `/settime HH:MM` | sets the delivery time, for example `/settime 14:30` |
| `/digest` | generates the digest immediately instead of waiting for the schedule |

### dev tests

The repository tests need a throwaway database and are skipped when it is not configured, so the name must contain `test`, cuz the fixtures truncate the tables.

```bash
createdb radar_test
psql "$TEST_DATABASE_URL" -f migrations/001_create_users.sql
psql "$TEST_DATABASE_URL" -f migrations/002_create_preferences.sql
psql "$TEST_DATABASE_URL" -f migrations/003_digest_disabled_by_default.sql

TEST_DATABASE_URL="postgresql://user:pass@localhost:5432/radar_test" go test -race ./...
```

## Docs for used libs

- go-telegram/bot: https://pkg.go.dev/github.com/go-telegram/bot
- MITRE ATT&CK CTI and STIX data: https://github.com/mitre/cti
- CWE research dataset: https://cwe.mitre.org/data/downloads.html
---
- pgx v5 driver: https://pkg.go.dev/github.com/jackc/pgx/v5
- River job queue: https://riverqueue.com/docs
- River migrations: https://riverqueue.com/docs/migrations
- sqlc (SQL to Go, regenerates `internal/infrastructure/postgres`): https://sqlc.dev/docs
- robfig/cron for river: https://pkg.go.dev/github.com/robfig/cron/v3
- golang.org/x/net/proxy: https://pkg.go.dev/golang.org/x/net/proxy
- golangci-lint: https://golangci-lint.run/docs/linters/factories/
- gofumpt: https://github.com/mvdan/gofumpt

## how it works

`cmd/bot` and `cmd/worker` are separate processes over one Postgres database, sharing `internal/bootstrap` for the pool, the proxied Telegram client and the ATT&CK graph plus CWE dataset parsed from `data/`. `internal/domain` holds plain structs and only time-of-day parsing; `internal/application` owns the use cases; `internal/infrastructure` is split per backing service (`postgres`, `telegram`, `mitre`, `cwe`); `internal/handler` is the conversation and keeps keyboard state in memory only; `internal/job` is the River side. Persistence is sqlc: `queries/` holds the SQL, `internal/infrastructure/postgres/*.sql.go` is generated from it, and `UserRepository` is the only thing the layers above see. The worker runs a River periodic job every minute; the tick selects enabled users whose `delivery_time` falls in the half-open window between the previous and the current UTC minute, then inserts one unique `send_digest` job per user, which regenerates the digest and pushes it over Telegram. Times are matched in UTC, so `/settime 18:30` means 18:30 UTC wherever the user is.

This architecture is not the best and needs a lot of fixes and improvements btw :)

## License

- MIT License (see `LICENSE`).

## Contributing

- Open an issue with reproduction steps or the desired feature.
- Keep changes lintable and small, prefer PRs that isolate one concern.
- Mention the migration you added and the commands you exercised when filing digest delivery bugs.
