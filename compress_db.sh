#!/bin/bash
# Compress ecd.db for upload to GitHub (max compression, ~25-30M).
# Usage: ./compress_db.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DB="$SCRIPT_DIR/ecd.db"
XZ="$SCRIPT_DIR/ecd.db.xz"

if [ ! -f "$DB" ]; then
    echo "Error: $DB not found. Build it first with ecd-build or python -m ecd_extract."
    exit 1
fi

echo "Compressing $DB ..."
echo "  Original: $(ls -lh "$DB" | awk '{print $5}')"

rm -f "$XZ"
xz -z9e -T0 -k "$DB"

echo "  Compressed: $(ls -lh "$XZ" | awk '{print $5}')"
echo "Done: $XZ"
