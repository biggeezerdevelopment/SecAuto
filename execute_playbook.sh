#!/bin/bash

# Execute the database_monitor automation/playbook
# This assumes the PostgreSQL integration is already configured for the client

CLIENT_ID="client_f1616270b0ef6513"
API_KEY="secauto-api-key-2024-07-14"

echo "Method 1: Execute database_monitor automation directly"
curl -X POST "http://localhost:9090/clients/${CLIENT_ID}/automations/database_monitor/execute" \
  -H "accept: */*" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "context": {
      "integration": "postgresql",
      "action": "monitor_tables"
    }
  }'

echo -e "\n\nMethod 2: Execute as a playbook with automation operation"
curl -X POST "http://localhost:9090/playbook" \
  -H "accept: */*" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [
      {
        "automation": "database_monitor",
        "context": {
          "integration": "postgresql",
          "action": "monitor_tables"
        }
      }
    ]
  }'