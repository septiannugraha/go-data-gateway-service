#!/bin/bash

echo "=========================================="
echo "n8n Webhook Test Script"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Webhook URLs
TEST_URL="http://localhost:5678/webhook-test/google-forms-webhook"
PROD_URL="http://localhost:5678/webhook/google-forms-webhook"

echo -e "${YELLOW}Testing n8n Webhook Endpoints${NC}"
echo ""

# Test 1: Valid passphrase (test mode)
echo -e "${GREEN}Test 1: Valid Registration (Test Mode)${NC}"
echo "URL: $TEST_URL"
echo "Note: Click 'Execute workflow' in n8n before running this!"
echo ""
curl -X POST $TEST_URL \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testuser@example.com",
    "name": "Test User",
    "passphrase": "spse2025"
  }' 2>/dev/null | jq . || echo "No JSON response"

echo ""
echo "----------------------------------------"
echo ""

# Test 2: Invalid passphrase (test mode)
echo -e "${GREEN}Test 2: Invalid Passphrase (Test Mode)${NC}"
echo "URL: $TEST_URL"
echo "Note: Click 'Execute workflow' in n8n again before running this!"
echo ""
read -p "Press Enter after clicking 'Execute workflow' in n8n..."
curl -X POST $TEST_URL \
  -H "Content-Type: application/json" \
  -d '{
    "email": "invalid@example.com",
    "name": "Invalid User",
    "passphrase": "wrong"
  }' 2>/dev/null | jq . || echo "No JSON response"

echo ""
echo "----------------------------------------"
echo ""

# Test 3: Production webhook
echo -e "${GREEN}Test 3: Production Webhook${NC}"
echo "URL: $PROD_URL"
echo "Note: Workflow must be activated for this to work!"
echo ""
curl -X POST $PROD_URL \
  -H "Content-Type: application/json" \
  -d '{
    "email": "production@example.com",
    "name": "Production User",
    "passphrase": "spse2025"
  }' 2>/dev/null | jq . || echo "No JSON response"

echo ""
echo "=========================================="
echo -e "${YELLOW}Testing Complete!${NC}"
echo ""
echo "📝 Notes:"
echo "1. Test webhooks require clicking 'Execute workflow' each time"
echo "2. Production webhooks require the workflow to be activated"
echo "3. Check n8n UI for execution results"
echo "4. Valid passphrase: spse2025"
echo ""
echo "🔗 n8n UI: http://localhost:5678"
echo "=========================================="