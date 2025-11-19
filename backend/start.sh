#!/bin/sh

set -e

/app/wait-for-it.sh db:5432 -- echo "Database is up"

# Run migrations
alembic upgrade head

# Create initial data
python -m app.init_db

# Start server
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
