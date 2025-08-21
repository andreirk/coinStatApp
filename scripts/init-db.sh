#!/bin/bash

# Initialize ClickHouse tables
# Usage: ./init-db.sh [clickhouse_host] [clickhouse_port]

CLICKHOUSE_HOST=${1:-localhost}
CLICKHOUSE_PORT=${2:-8123}

echo "Initializing ClickHouse tables on ${CLICKHOUSE_HOST}:${CLICKHOUSE_PORT}..."

cat scripts/init-storage.sql | curl -s "http://${CLICKHOUSE_HOST}:${CLICKHOUSE_PORT}/" --data-binary @-

if [ $? -eq 0 ]; then
  echo "Database initialization completed successfully!"
else
  echo "Failed to initialize database."
  exit 1
fi
