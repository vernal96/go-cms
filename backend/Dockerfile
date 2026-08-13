FROM golang:1.26.1-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server \
    ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/console \
    ./cmd/console

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S cms \
    && adduser -S -G cms cms

WORKDIR /app

RUN mkdir -p var/log var/files/public var/files/private \
    && chown -R cms:cms /app

COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/console /usr/local/bin/console

USER cms

EXPOSE 8080

CMD ["/usr/local/bin/server"]
