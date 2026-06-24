# Development vertical slice

This temporary note documents the first CMS vertical slice.

Run PostgreSQL:

```bash
docker compose up -d postgres
```

Run the app:

```bash
go mod tidy
go run ./cmd/app
```

Check the core site info route:

```bash
curl http://localhost:8080/_cms/site
curl -H 'Host: example.com' http://localhost:8080/_cms/site
```
