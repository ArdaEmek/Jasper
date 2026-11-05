#!/bin/sh
set -eu

case "${1:-serve}" in
  serve)
    shift
    exec /app/main "$@"
    ;;
  migrate)
    shift
    export DBMATE_MIGRATIONS_DIR="${DBMATE_MIGRATIONS_DIR:-/app/db/migrations}"
    exec dbmate "$@"
    ;;
  *)
    exec "$@"
    ;;
esac
