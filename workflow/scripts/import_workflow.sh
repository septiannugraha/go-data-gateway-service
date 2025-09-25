#!/bin/bash

# Import workflow into n8n
# This script imports the user onboarding workflow via n8n API

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
N8N_URL=${N8N_URL:-"http://localhost:5678"}
WORKFLOW_FILE="../n8n/workflows/user_onboarding.json"

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  n8n Workflow Import Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if n8n is running
echo -e "${YELLOW}Checking n8n status...${NC}"
if ! curl -f "$N8N_URL/healthz" >/dev/null 2>&1; then
    echo -e "${RED}❌ n8n is not accessible at $N8N_URL${NC}"
    echo "Please ensure n8n is running: docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✅ n8n is running${NC}"

# Check if workflow file exists
if [ ! -f "$SCRIPT_DIR/$WORKFLOW_FILE" ]; then
    echo -e "${RED}❌ Workflow file not found: $WORKFLOW_FILE${NC}"
    exit 1
fi

# Read workflow JSON
WORKFLOW_JSON=$(cat "$SCRIPT_DIR/$WORKFLOW_FILE")

# Check if basic auth is enabled
if [ ! -z "$N8N_BASIC_AUTH_USER" ] && [ ! -z "$N8N_BASIC_AUTH_PASSWORD" ]; then
    echo -e "${YELLOW}Using basic authentication...${NC}"
    AUTH_HEADER="Authorization: Basic $(echo -n $N8N_BASIC_AUTH_USER:$N8N_BASIC_AUTH_PASSWORD | base64)"
else
    AUTH_HEADER=""
fi

# Import workflow
echo ""
echo -e "${YELLOW}Importing workflow...${NC}"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$N8N_URL/rest/workflows" \
    -H "Content-Type: application/json" \
    ${AUTH_HEADER:+-H "$AUTH_HEADER"} \
    -d "$WORKFLOW_JSON")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✅ Workflow imported successfully${NC}"

    # Extract workflow ID from response
    WORKFLOW_ID=$(echo "$BODY" | grep -o '"id":"[^"]*' | grep -o '[^"]*$' || echo "")

    if [ ! -z "$WORKFLOW_ID" ]; then
        echo -e "${BLUE}Workflow ID: $WORKFLOW_ID${NC}"

        # Activate the workflow
        echo ""
        echo -e "${YELLOW}Activating workflow...${NC}"

        ACTIVATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$N8N_URL/rest/workflows/$WORKFLOW_ID" \
            -H "Content-Type: application/json" \
            ${AUTH_HEADER:+-H "$AUTH_HEADER"} \
            -d '{"active": true}')

        ACTIVATE_CODE=$(echo "$ACTIVATE_RESPONSE" | tail -n 1)

        if [ "$ACTIVATE_CODE" = "200" ]; then
            echo -e "${GREEN}✅ Workflow activated${NC}"
        else
            echo -e "${YELLOW}⚠️  Could not activate workflow automatically${NC}"
            echo "Please activate it manually in the n8n UI"
        fi
    fi
else
    echo -e "${RED}❌ Failed to import workflow (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
    echo ""
    echo -e "${YELLOW}Troubleshooting:${NC}"
    echo "1. Check if n8n is fully initialized"
    echo "2. Try importing manually via UI"
    echo "3. Check n8n logs: docker-compose logs n8n"
    exit 1
fi

# Display summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Import Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "1. Access n8n UI: $N8N_URL"
echo "2. Go to Workflows → User Onboarding"
echo "3. Configure credentials:"
echo "   - Gmail OAuth2 for email sending"
echo "   - Any required API credentials"
echo "4. Test the workflow with: ./test.sh"
echo ""
echo -e "${BLUE}Webhook URL:${NC}"
echo "$N8N_URL/webhook/google-forms-webhook"
echo ""
echo -e "${YELLOW}Configure this URL in FormLinker on your Google Form${NC}"
echo ""
echo -e "${GREEN}Workflow is ready to use! 🚀${NC}"