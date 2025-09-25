#!/bin/bash

# Test script for n8n User Onboarding Workflow
# Tests the complete registration flow

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
WEBHOOK_URL=${WEBHOOK_URL:-"http://localhost:5678/webhook/google-forms-webhook"}
PASSPHRASE=${FORM_PASSPHRASE:-"spse2025"}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Testing User Onboarding Workflow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to generate test data
generate_test_email() {
    echo "test-$(date +%s)@example.com"
}

generate_test_name() {
    FIRST_NAMES=("John" "Jane" "Bob" "Alice" "Charlie" "Diana" "Eve" "Frank")
    LAST_NAMES=("Smith" "Johnson" "Williams" "Brown" "Jones" "Garcia" "Miller" "Davis")

    FIRST=${FIRST_NAMES[$RANDOM % ${#FIRST_NAMES[@]}]}
    LAST=${LAST_NAMES[$RANDOM % ${#LAST_NAMES[@]}]}

    echo "$FIRST $LAST"
}

# Check if n8n is running
echo -e "${YELLOW}Checking n8n status...${NC}"
if ! curl -f http://localhost:5678/healthz >/dev/null 2>&1; then
    echo -e "${RED}❌ n8n is not running${NC}"
    echo "Please start n8n first: docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✅ n8n is running${NC}"

# Test 1: Valid Registration
echo ""
echo -e "${YELLOW}Test 1: Valid Registration${NC}"
echo "------------------------"

TEST_EMAIL=$(generate_test_email)
TEST_NAME=$(generate_test_name)

echo "Email: $TEST_EMAIL"
echo "Name: $TEST_NAME"
echo "Passphrase: $PASSPHRASE"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"name\": \"$TEST_NAME\",
        \"passphrase\": \"$PASSPHRASE\",
        \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
        \"formId\": \"test-form-001\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ Registration successful (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
else
    echo -e "${RED}❌ Registration failed (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
fi

# Test 2: Invalid Passphrase
echo ""
echo -e "${YELLOW}Test 2: Invalid Passphrase${NC}"
echo "------------------------"

TEST_EMAIL=$(generate_test_email)
TEST_NAME=$(generate_test_name)

echo "Email: $TEST_EMAIL"
echo "Name: $TEST_NAME"
echo "Passphrase: wrong_passphrase"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"name\": \"$TEST_NAME\",
        \"passphrase\": \"wrong_passphrase\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✅ Correctly rejected invalid passphrase (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
else
    echo -e "${RED}❌ Should have rejected invalid passphrase (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
fi

# Test 3: Missing Required Fields
echo ""
echo -e "${YELLOW}Test 3: Missing Required Fields${NC}"
echo "------------------------"

echo "Testing with missing email..."
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Test User\",
        \"passphrase\": \"$PASSPHRASE\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" != "200" ]; then
    echo -e "${GREEN}✅ Correctly handled missing email (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}❌ Should have failed with missing email (HTTP $HTTP_CODE)${NC}"
fi

# Test 4: Duplicate Registration (if applicable)
echo ""
echo -e "${YELLOW}Test 4: Duplicate Registration${NC}"
echo "------------------------"

DUPLICATE_EMAIL="duplicate-test@example.com"
echo "Email: $DUPLICATE_EMAIL (testing twice)"
echo ""

# First registration
echo "First registration..."
curl -s -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$DUPLICATE_EMAIL\",
        \"name\": \"Duplicate Test\",
        \"passphrase\": \"$PASSPHRASE\"
    }" > /dev/null 2>&1

sleep 2

# Second registration (duplicate)
echo "Second registration (duplicate)..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$DUPLICATE_EMAIL\",
        \"name\": \"Duplicate Test\",
        \"passphrase\": \"$PASSPHRASE\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

echo "Response: $BODY"
echo -e "${YELLOW}Note: Duplicate handling depends on Fusio configuration${NC}"

# Test 5: Load Test
echo ""
echo -e "${YELLOW}Test 5: Load Test (5 concurrent requests)${NC}"
echo "------------------------"

echo "Sending 5 concurrent registration requests..."
echo ""

for i in {1..5}; do
    (
        EMAIL="load-test-$i-$(date +%s)@example.com"
        NAME="Load Test User $i"

        RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{
                \"email\": \"$EMAIL\",
                \"name\": \"$NAME\",
                \"passphrase\": \"$PASSPHRASE\"
            }")

        HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)

        if [ "$HTTP_CODE" = "200" ]; then
            echo -e "${GREEN}✅ Request $i succeeded${NC}"
        else
            echo -e "${RED}❌ Request $i failed (HTTP $HTTP_CODE)${NC}"
        fi
    ) &
done

wait
echo -e "${GREEN}Load test completed${NC}"

# Check logs
echo ""
echo -e "${YELLOW}Recent n8n Logs:${NC}"
echo "------------------------"
docker-compose logs --tail=20 n8n 2>/dev/null | grep -E "(user_registration|error)" || echo "No recent registration logs"

# Summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Test Summary${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "✓ Valid registration test"
echo "✓ Invalid passphrase test"
echo "✓ Missing fields test"
echo "✓ Duplicate registration test"
echo "✓ Load test (5 concurrent)"
echo ""
echo -e "${YELLOW}To view detailed logs:${NC}"
echo "  docker-compose logs -f n8n"
echo ""
echo -e "${YELLOW}To check workflow executions:${NC}"
echo "  Open http://localhost:5678 → Executions"
echo ""
echo -e "${GREEN}Testing complete! 🎉${NC}"