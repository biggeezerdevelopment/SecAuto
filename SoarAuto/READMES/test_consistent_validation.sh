#!/bin/bash
# Test script to verify both execution paths have consistent validation

BASE_URL="http://localhost:9090"
CLIENT_ID="client_f1616270b0ef6513"  # This client has disabled postgresql integration
API_KEY="secauto-api-key-2024-07-14"

echo "Testing Consistent Integration Validation"
echo "========================================="
echo ""
echo "Client: $CLIENT_ID (PostgreSQL integration is DISABLED)"
echo ""

echo "1. Testing direct integration execution (should be blocked):"
echo "   /clients/.../integrations/.../execute"
echo ""

curl -X POST "$BASE_URL/clients/$CLIENT_ID/integrations/postgresql/execute" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"function": "test_connection", "params": {}}' \
  -s | jq '.'

echo ""
echo "2. Testing run_i execution (should now also be blocked):"
echo "   /playbook with run_i"
echo ""

curl -X POST "$BASE_URL/playbook" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "context": {"client_id": "'$CLIENT_ID'"},
    "playbook": [
      {
        "run": "database_monitor",
        "run_i": "postgresql"
      }
    ]
  }' \
  -s | jq '.'

echo ""
echo "Expected Results:"
echo "- Both should return 'Integration is disabled for this client'"
echo "- Both should have success: false"
echo ""
echo "If run_i still succeeds, the fix needs to be deployed to the server"