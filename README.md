# Go CMS

The repository is a monorepo with two independent applications:

```text
cms/
├── backend/        # Go API and backend infrastructure
├── frontend-admin/ # Vue administration interface
└── compose.yaml    # Shared development orchestration
```

The frontend communicates with the backend only over HTTP. Browser requests use relative `/api` paths; the Vite development server proxies them to `ADMIN_API_TARGET`.

## Requirements

- GNU Make
- Docker with a recent Docker Compose version that supports `--wait`

Go as declared in `backend/go.mod` and Node.js 24 or newer are only required
when running backend or frontend checks directly on the host.

## Development

Build, initialize, and start the complete development environment:

```bash
make up
```

Running `make` without a target does the same thing. The command:

- creates `.env` from `.env.example` when it is missing and never overwrites an
  existing file;
- builds the backend and administration images;
- starts and waits for PostgreSQL, Kafka, RabbitMQ, Loki, and Grafana;
- applies all database migrations and the development seeds;
- starts the backend and administration frontend and waits for their
  healthchecks.

After startup, the services are available at these default addresses:

- administration interface: `http://localhost:5173`;
- backend API: `http://localhost:8080`;
- Grafana: `http://localhost:3000`.

Sign in to the administration interface with the development seed account:

```bash
login: admin
password: admin-dev-only-2026
```

`make up` is idempotent and can be used again after pulling new changes. It
reapplies pending migrations and seeds without deleting persistent data.

Useful development commands:

```bash
make ps    # show the status of every service
make logs  # follow logs from every service
make build # rebuild application images
make down  # stop containers without deleting persistent volumes
make help  # show all Make targets
```

## Checks

```bash
go -C backend test ./...
npm --prefix frontend-admin ci
npm --prefix frontend-admin test
npm --prefix frontend-admin run build
docker compose --env-file .env.example config --quiet
```
