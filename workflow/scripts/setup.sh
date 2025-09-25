#!/bin/bash

# Setup script for n8n User Onboarding Workflow
# This script initializes the workflow environment

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
WORKFLOW_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  n8n Workflow Setup Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to generate random string
generate_random_string() {
    openssl rand -hex 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
}

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command_exists docker; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✅ Docker found${NC}"

if ! command_exists docker-compose; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose found${NC}"

# Check if .env exists
echo ""
echo -e "${YELLOW}Setting up environment configuration...${NC}"

cd "$WORKFLOW_DIR"

if [ -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file already exists${NC}"
    read -p "Do you want to backup and create new? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cp .env .env.backup.$(date +%Y%m%d_%H%M%S)
        echo -e "${GREEN}✅ Backed up existing .env${NC}"
    else
        echo -e "${BLUE}Using existing .env file${NC}"
    fi
else
    cp .env.example .env
    echo -e "${GREEN}✅ Created .env from .env.example${NC}"
fi

# Generate encryption key if not set
if grep -q "your-32-character-encryption-key-here" .env; then
    echo -e "${YELLOW}Generating encryption key...${NC}"
    ENCRYPTION_KEY=$(generate_random_string)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/your-32-character-encryption-key-here/$ENCRYPTION_KEY/" .env
    else
        sed -i "s/your-32-character-encryption-key-here/$ENCRYPTION_KEY/" .env
    fi
    echo -e "${GREEN}✅ Generated encryption key${NC}"
fi

# Prompt for critical configuration
echo ""
echo -e "${YELLOW}Please provide the following configuration:${NC}"
echo -e "${YELLOW}(Press Enter to skip and configure later)${NC}"
echo ""

# Fusio configuration
read -p "Fusio API URL [http://localhost:8000]: " FUSIO_URL
FUSIO_URL=${FUSIO_URL:-http://localhost:8000}

read -p "Fusio API Key: " FUSIO_KEY
if [ ! -z "$FUSIO_KEY" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|FUSIO_API_URL=.*|FUSIO_API_URL=$FUSIO_URL|" .env
        sed -i '' "s/FUSIO_API_KEY=.*/FUSIO_API_KEY=$FUSIO_KEY/" .env
    else
        sed -i "s|FUSIO_API_URL=.*|FUSIO_API_URL=$FUSIO_URL|" .env
        sed -i "s/FUSIO_API_KEY=.*/FUSIO_API_KEY=$FUSIO_KEY/" .env
    fi
fi

# Gmail configuration
echo ""
read -p "Gmail address for sending emails: " GMAIL_USER
if [ ! -z "$GMAIL_USER" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/GMAIL_USER=.*/GMAIL_USER=$GMAIL_USER/" .env
    else
        sed -i "s/GMAIL_USER=.*/GMAIL_USER=$GMAIL_USER/" .env
    fi
fi

read -s -p "Gmail App Password (hidden): " GMAIL_PASS
echo
if [ ! -z "$GMAIL_PASS" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/GMAIL_APP_PASSWORD=.*/GMAIL_APP_PASSWORD=$GMAIL_PASS/" .env
    else
        sed -i "s/GMAIL_APP_PASSWORD=.*/GMAIL_APP_PASSWORD=$GMAIL_PASS/" .env
    fi
fi

# Form passphrase
echo ""
read -p "Registration passphrase [spse2025]: " PASSPHRASE
PASSPHRASE=${PASSPHRASE:-spse2025}
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/FORM_PASSPHRASE=.*/FORM_PASSPHRASE=$PASSPHRASE/" .env
else
    sed -i "s/FORM_PASSPHRASE=.*/FORM_PASSPHRASE=$PASSPHRASE/" .env
fi

# Create necessary directories
echo ""
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p n8n/workflows n8n/credentials services scripts docs
echo -e "${GREEN}✅ Directories created${NC}"

# Copy services to n8n custom directory (for function nodes)
echo ""
echo -e "${YELLOW}Setting up custom modules...${NC}"
mkdir -p n8n_data/custom
cp services/*.js n8n_data/custom/ 2>/dev/null || true
echo -e "${GREEN}✅ Custom modules prepared${NC}"

# Check if network exists
echo ""
echo -e "${YELLOW}Checking Docker network...${NC}"
if ! docker network ls | grep -q "data-gateway-service_default"; then
    echo -e "${YELLOW}Creating data-gateway-service_default network...${NC}"
    docker network create data-gateway-service_default
    echo -e "${GREEN}✅ Network created${NC}"
else
    echo -e "${GREEN}✅ Network already exists${NC}"
fi

# Start services
echo ""
echo -e "${YELLOW}Starting n8n services...${NC}"
docker-compose up -d

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to start...${NC}"
sleep 10

# Check service health
echo ""
echo -e "${YELLOW}Checking service health...${NC}"

if curl -f http://localhost:5678/healthz >/dev/null 2>&1; then
    echo -e "${GREEN}✅ n8n is running${NC}"
else
    echo -e "${YELLOW}⚠️  n8n is still starting, please wait...${NC}"
fi

if docker exec n8n-postgres pg_isready >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL is running${NC}"
else
    echo -e "${YELLOW}⚠️  PostgreSQL is still starting...${NC}"
fi

# Display access information
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}n8n Interface:${NC} http://localhost:5678"
echo -e "${BLUE}Webhook URL:${NC} http://localhost:5678/webhook/google-forms-webhook"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Access n8n UI at http://localhost:5678"
echo "2. Import workflow: n8n/workflows/user_onboarding.json"
echo "3. Configure Gmail OAuth2 credentials in n8n"
echo "4. Set up FormLinker in your Google Form"
echo "5. Test the workflow with ./scripts/test.sh"
echo ""
echo -e "${YELLOW}To view logs:${NC}"
echo "  docker-compose logs -f n8n"
echo ""
echo -e "${YELLOW}To stop services:${NC}"
echo "  docker-compose down"
echo ""
echo -e "${GREEN}Happy automating! 🚀${NC}"