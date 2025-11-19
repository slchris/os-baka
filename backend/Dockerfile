FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

COPY wait-for-it.sh /usr/local/bin/wait-for-it.sh
COPY start.sh /app/start.sh
RUN chmod +x /usr/local/bin/wait-for-it.sh /app/start.sh
