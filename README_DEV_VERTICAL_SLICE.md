# Development vertical slice

This temporary note documents the first CMS vertical slice.

Run the development infrastructure:

```bash
docker compose up -d postgres redis kafka
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
curl "http://localhost:8080/_cms/resource?path=/"
```

Configuration can be overridden with:

- `GO_CMS_HTTP_ADDR`
- `GO_CMS_DATABASE_DSN`
- `GO_CMS_REDIS_ADDR`, `GO_CMS_REDIS_PASSWORD`, `GO_CMS_REDIS_DB`
- `GO_CMS_STORAGE_LOCAL_ROOT`
- `GO_CMS_KAFKA_BROKERS`, `GO_CMS_KAFKA_TOPIC`, `GO_CMS_KAFKA_GROUP_ID`
