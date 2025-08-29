#!/bin/bash
# Test script to verify enhanced logging shows config details

echo "Testing Enhanced Integration Config Logging"
echo "==========================================="
echo ""
echo "This test will make an API call that triggers config retrieval"
echo "and show the enhanced logging output."
echo ""

# Configuration
BASE_URL="${SECAUTO_URL:-http://localhost:8080}"
CLIENT_ID="${CLIENT_ID:-client_a4fba03c29631889}"
INTEGRATION="${INTEGRATION:-postgresql}"

echo "Configuration:"
echo "  BASE_URL: $BASE_URL"
echo "  CLIENT_ID: $CLIENT_ID"
echo "  INTEGRATION: $INTEGRATION"
echo ""

# Make the API call to get integration config
echo "Making API call to retrieve integration config..."
echo ""

curl -s -X GET "$BASE_URL/api/v1/clients/$CLIENT_ID/integrations/$INTEGRATION/config" | jq '.'

echo ""
echo "==========================================="
echo "CHECK SERVER LOGS for enhanced output!"
echo "==========================================="
echo ""
echo "You should now see detailed logging in the server output including:"
echo ""
echo "1. Cache miss/hit information with cache key"
echo "2. Config retrieval from database or file:"
echo "   - client_id"
echo "   - integration name"
echo "   - enabled status"
echo "   - has_config: true/false"
echo "   - has_credentials: true/false"
echo "   - config_keys: [list of config keys]"
echo "   - credential_keys: [list of credential keys]"
echo ""
echo "3. If caching occurs:"
echo "   - Successfully cached config"
echo "   - cache_key"
echo "   - ttl_seconds"
echo ""
echo "4. When retrieved for use:"
echo "   - Retrieved client integration config"
echo "   - All the above fields in 'extra' JSON field"
echo ""
echo "Example log entry should look like:"
echo '{"timestamp":"...","level":"DEBUG","message":"Retrieved config from database","component":"system","extra":{"client_id":"...","config_keys":["host","port"],"credential_keys":["username","password"],"enabled":true,"has_config":true,"has_credentials":true,"integration":"postgresql"}}'