#!/bin/bash

set -euo pipefail

ACCESS_DB_PATH="/opt/doorbot2/access.sqlite"
DATE=$(date +%Y%m%d)
BACKUP_FILE="/var/backups/access-$DATE.sql.gz"

if [ ! -f "$ACCESS_DB_PATH" ]; then
    exit 0
fi

sqlite3 "$ACCESS_DB_PATH" .dump | gzip > "$BACKUP_FILE"
aws s3 cp "$BACKUP_FILE" s3://fcfl-backups/fcfl-access/
rm "$BACKUP_FILE"
