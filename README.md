# Date Chooser

A self-hosted, account-free meeting scheduler (Doodle/Rallly-style). Create a poll with
candidate date/time slots, share a participant link and a secret admin link, collect
Yes/No/Maybe votes with comments, and see a results grid with the best slot highlighted.
No accounts, no external services — a single Go binary backed by SQLite.

## Run with Docker (recommended)

```bash
docker build -t datechooser .
docker run -d --name datechooser \
  -p 8080:8080 \
  -v datechooser-data:/data \
  datechooser
```

Then open http://localhost:8080

Data persists in the `datechooser-data` named volume across restarts.

## Run locally with Go

Requires Go 1.25+.

The default `DB_PATH` (`/data/datechooser.db`) assumes a Docker-style `/data` volume, which
won't exist on a plain dev machine — set it to a local path when running outside Docker:

```bash
DB_PATH=./datechooser.db go run ./cmd/server
```

Override the port too if needed:

```bash
PORT=3000 DB_PATH=./datechooser.db go run ./cmd/server
```

Then open http://localhost:8080 (or whatever `PORT` you set).

## Configuration

| Env var | Default | Description |
|---------|---------|--------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `/data/datechooser.db` | Path to the SQLite database file — override to a local path when not running in Docker |

## Health check

```bash
curl http://localhost:8080/healthz
```

Returns `200 OK` once the server is up and the database is reachable.

## Development

```bash
go build ./...      # build
go vet ./...         # vet
go test ./... -count=1   # run tests
gofmt -l .           # check formatting
```

## How it works

1. **Create a poll** at `/` — title, optional description, and a set of candidate slots
   (all-day dates or specific date+time ranges).
2. You're given two links: a **participant link** to share with your group, and a **secret
   admin link** — keep that one private.
3. Participants open the participant link (no login), enter a name, vote Yes/No/Maybe per
   slot, and can add a comment. Revisiting the same link later lets them change their vote.
4. Both links show a **results grid** — everyone's answers, per-slot tallies, and the
   best-fitting slot(s) highlighted.
5. The admin link additionally lets the organizer edit the poll (title/description/slots),
   remove a participant's response, or delete the whole poll.
