#!/bin/bash

# Configure PostgreSQL integration for the client
# Replace the database credentials with your actual PostgreSQL database details

CLIENT_ID="client_f1616270b0ef6513"
API_KEY="secauto-api-key-2024-07-14"

curl -X POST "http://localhost:9090/clients/${CLIENT_ID}/integrations/postgresql/config" \
  -H "accept: */*" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "postgresql",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "your_database",
      "username": "your_username", 
      "password": "your_password",
      "ssl_mode": "prefer"
    }
  }'