#!/bin/bash
# Creates per-service databases and runs their migration scripts.
# Reads POSTGRES_MULTIPLE_DATABASES env var (comma-separated list of DB names).

set -e
set -u

if [ -z "${POSTGRES_MULTIPLE_DATABASES:-}" ]; then
  echo "POSTGRES_MULTIPLE_DATABASES is not set, skipping"
  exit 0
fi

IFS=',' read -ra DATABASES <<< "$POSTGRES_MULTIPLE_DATABASES"
for db in "${DATABASES[@]}"; do
  db=$(echo "$db" | xargs) # trim whitespace
  echo "Creating database: $db"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    SELECT 'CREATE DATABASE $db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db')\gexec
EOSQL

  # Run migrations for this database if a matching directory exists.
  # Directory name is the suffix after "connect_" (e.g. connect_auth -> auth).
  suffix="${db#connect_}"
  migdir="/docker-entrypoint-initdb.d/$suffix"
  if [ -d "$migdir" ]; then
    echo "Running migrations for $db from $migdir"
    for f in "$migdir"/*.sql; do
      [ -f "$f" ] || continue
      echo "  Applying $f"
      psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" -f "$f"
    done
  fi
done
