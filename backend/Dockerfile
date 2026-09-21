FROM golang:1.26-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server
FROM alpine:3.22
RUN adduser -D -u 10001 cms
WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server
RUN mkdir -p /app/var && chown -R cms:cms /app
USER cms
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
