#!/bin/bash
# Test script to verify client integration delete functionality

BASE_URL="http://localhost:9090"
CLIENT_ID="client_f1616270b0ef6513"  
INTEGRATION="postgresql"
API_KEY="secauto-api-key-2024-07-14"

echo "Testing Client Integration Delete Functionality"
echo "=============================================="
echo ""
echo "Client: $CLIENT_ID"
echo "Integration: $INTEGRATION"
echo ""

echo "1. First, let's see the current integration config:"
curl -X GET "$BASE_URL/clients/$CLIENT_ID/integrations/$INTEGRATION/config" \
  -H "X-API-Key: $API_KEY" \
  -s | jq '.'

echo ""
echo "2. Now deleting the integration config:"
curl -X DELETE "$BASE_URL/clients/$CLIENT_ID/integrations/$INTEGRATION/config" \
  -H "X-API-Key: $API_KEY" \
  -s | jq '.'

echo ""
echo "3. Verify it's deleted (should return 404 or not found):"
curl -X GET "$BASE_URL/clients/$CLIENT_ID/integrations/$INTEGRATION/config" \
  -H "X-API-Key: $API_KEY" \
  -s | jq '.'

echo ""
echo "4. Check client integrations list (should not include postgresql):"
curl -X GET "$BASE_URL/clients/$CLIENT_ID/integrations" \
  -H "X-API-Key: $API_KEY" \
  -s | jq '.integrations[] | {name: .name, enabled: .enabled}'

echo ""
echo "Expected Results:"
echo "- Step 2 should return success: true, message about deletion"
echo "- Step 3 should return error/not found"
echo "- Step 4 should not list postgresql integration"
echo ""
echo "Database & Cache Effects:"
echo "- Record deleted from client_integration_configs table"
echo "- Redis cache key removed: client:$CLIENT_ID:integration:$INTEGRATION"