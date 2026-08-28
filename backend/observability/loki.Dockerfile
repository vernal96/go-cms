FROM busybox:1.37-musl AS tools

FROM grafana/loki:3.7.3

COPY --from=tools /bin/wget /usr/local/bin/wget
