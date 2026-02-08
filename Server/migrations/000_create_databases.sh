#!/bin/bash
# Create per-service databases within the same PostgreSQL instance.
# This script runs once during initial container startup via docker-entrypoint-initdb.d.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE connect_auth;
    CREATE DATABASE connect_users;
    CREATE DATABASE connect_chat;
    CREATE DATABASE connect_messages;
    CREATE DATABASE connect_matching;
EOSQL
