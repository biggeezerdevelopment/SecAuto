#!/bin/bash
# Script to recreate a test integration config after deletion

BASE_URL="http://localhost:9090"
CLIENT_ID="client_f1616270b0ef6513"
INTEGRATION="postgresql"
API_KEY="secauto-api-key-2024-07-14"

echo "Recreating Test Integration Config"
echo "=================================="
echo ""

curl -X PUT "$BASE_URL/clients/$CLIENT_ID/integrations/$INTEGRATION/config" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "enabled": false,
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "testdb", 
    "username": "testuser",
    "password": "testpassword",
    "ssl_mode": "disable"
  }
}' | jq '.'

echo ""
echo "Integration config recreated for testing purposes"