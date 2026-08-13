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

- Go as declared in `backend/go.mod`
- Node.js 24 or newer
- Docker with Docker Compose

## Development

Create the shared local environment file:

```bash
cp .env.example .env
```

Start the backend server, administration frontend, and infrastructure services:

```bash
docker compose up --build
```

The administration interface is available at `http://localhost:5173` by default, and the backend API is available at `http://localhost:8080`. The backend server starts automatically with its container.

## Checks

```bash
go -C backend test ./...
npm --prefix frontend-admin ci
npm --prefix frontend-admin test
npm --prefix frontend-admin run build
docker compose --env-file .env.example config --quiet
```
