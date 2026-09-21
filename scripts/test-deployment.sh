#!/bin/sh
# Test the current source in a fresh directory, with new Compose volumes.
set -eu
command -v python3 >/dev/null 2>&1 || { echo 'Python 3 is required' >&2; exit 1; }
source_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/cms-deployment-XXXXXX")
export COMPOSE_PROJECT_NAME="cms-test-$(date +%s)-$$"
export SERVER_PORT="${DEPLOYMENT_TEST_PORT:-18080}"
export BASE_URL="http://localhost:$SERVER_PORT"
cleanup() {
  result=$?
  trap - EXIT HUP INT TERM
  if [ -f "$test_dir/.env" ]; then
    if [ "$result" -ne 0 ]; then
      (cd "$test_dir" && docker compose --env-file .env logs --no-color --tail=100) || true
    fi
    (cd "$test_dir" && docker compose --env-file .env down --volumes --remove-orphans) || true
  fi
  rm -rf "$test_dir"
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
cd "$source_dir"
git ls-files -z --cached --others --exclude-standard | tar --null -T - -cf - | tar -xf - -C "$test_dir"
cd "$test_dir"
make up
./scripts/smoke.sh --exercise --state "$test_dir/smoke-state.json"
# A failure reported to the file logger must also be visible in Docker logs.
if docker compose --env-file .env run --rm --no-deps \
  -e POSTGRES_PASSWORD=invalid-smoke-password \
  -e LOGGER_FILE_PATH=/tmp/startup-failure.log server > "$test_dir/startup-failure.log" 2>&1; then
  echo 'FAIL invalid database credentials unexpectedly started the server' >&2
  exit 1
fi
python3 - "$test_dir/startup-failure.log" <<'PY'
import pathlib, sys
message = pathlib.Path(sys.argv[1]).read_text()
if 'ping database connector' not in message:
    raise SystemExit('FAIL database startup failure was not reported to container logs: ' + message)
print('PASS database startup failure is visible in container logs')
PY
docker compose --env-file .env down
# No -v: the following check must read the data written by the first instance.
docker compose --env-file .env up --detach --wait --wait-timeout 180
./scripts/smoke.sh --verify --state "$test_dir/smoke-state.json"
echo 'PASS clean deployment and persistence across container recreation'
