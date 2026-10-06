#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
cd "$root"
export PATH="/usr/local/go/bin:$PATH"
air="$(go env GOPATH)/bin/air"
pid_file="$root/tmp/dev-air.pid"

running() {
	[[ -f "$pid_file" ]] || return 1
	IFS= read -r pid < "$pid_file"
	[[ "$pid" =~ ^[0-9]+$ ]] || return 1
	[[ "$(ps -p "$pid" -o command= 2>/dev/null || true)" == "$air -c .air.toml" ]]
}

case "${1:-}" in
start)
	if running; then
		docker compose up -d postgres redis
		printf 'Go API is already running (PID %s).\n' "$pid"
		exit 0
	fi
	if [[ ! -x "$air" ]]; then
		printf 'Install Air first: PATH="/usr/local/go/bin:$PATH" go install github.com/air-verse/air@v1.52.3\n' >&2
		exit 1
	fi
	docker compose stop api
	if command -v lsof >/dev/null && lsof -nP -iTCP:8080 -sTCP:LISTEN >/dev/null; then
		printf 'Port 8080 is already in use. Stop the other Go server first.\n' >&2
		exit 1
	fi
	docker compose up -d postgres redis
	mkdir -p "$root/tmp"
	umask 077
	nohup "$root/dev.sh" serve > "$root/tmp/dev.log" 2>&1 < /dev/null &
	pid=$!
	printf '%s\n' "$pid" > "$pid_file"
	for ((i = 0; i < 30; i++)); do
		if running && curl -fsS -o /dev/null http://localhost:8080/ready; then
			printf 'Go API ready at http://localhost:8080 (auto-reload on). Logs: tmp/dev.log\n'
			exit 0
		fi
		if ! kill -0 "$pid" 2>/dev/null; then
			break
		fi
		sleep 1
	done
	printf 'Go API did not become ready. Check tmp/dev.log.\n' >&2
	exit 1
	;;
serve)
	set -a
	. "$root/.env.example"
	set +a
	exec "$air" -c .air.toml
	;;
stop)
	if running; then
		kill -TERM "$pid"
		for ((i = 0; i < 30; i++)); do
			running || break
			sleep 0.1
		done
		if running; then
			printf 'Could not stop the Go API (PID %s).\n' "$pid" >&2
			exit 1
		fi
		printf 'Go API stopped.\n'
	fi
	if [[ -f "$pid_file" ]]; then
		rm -f "$pid_file"
	fi
	docker compose stop postgres redis
	;;
status)
	if running; then
		printf 'Go API running with auto-reload (PID %s).\n' "$pid"
	else
		printf 'Go API is not running via dev.sh.\n'
	fi
	docker compose ps postgres redis
	;;
*)
	printf 'Usage: ./dev.sh {start|stop|status}\n' >&2
	exit 1
	;;
esac
