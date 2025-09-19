#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Go Data Gateway Service Starter${NC}\n"

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from template...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✅ Created .env file. Please edit it with your credentials.${NC}"
    echo -e "${YELLOW}📝 Opening .env for editing...${NC}\n"
    ${EDITOR:-nano} .env
fi

# Check for BigQuery credentials
if [ ! -f ./credentials/bigquery-key.json ]; then
    echo -e "${YELLOW}⚠️  BigQuery credentials not found.${NC}"
    echo -e "   Please place your service account JSON at: ./credentials/bigquery-key.json"
    echo -e "   ${YELLOW}Creating credentials directory...${NC}"
    mkdir -p credentials
    echo -e "   ${GREEN}✅ Directory created. Add your bigquery-key.json file there.${NC}\n"
fi

# Menu
echo "Select an option:"
echo "1) Start with Docker Compose (recommended)"
echo "2) Start with Go run (development)"
echo "3) Build Docker image only"
echo "4) Stop all services"
echo "5) View logs"
echo "6) Run tests"

read -p "Enter your choice (1-6): " choice

case $choice in
    1)
        echo -e "\n${GREEN}Starting services with Docker Compose...${NC}"
        docker-compose up -d
        echo -e "\n${GREEN}✅ Services started!${NC}"
        echo -e "\nAccess points:"
        echo -e "  • Fusio Gateway: ${GREEN}http://localhost${NC}"
        echo -e "  • Go Service: ${GREEN}http://localhost:8080${NC}"
        echo -e "  • Grafana: ${GREEN}http://localhost:3000${NC} (admin/admin)"
        echo -e "  • Prometheus: ${GREEN}http://localhost:9090${NC}"
        echo -e "\nTest with:"
        echo -e "  ${YELLOW}curl -H 'X-API-Key: demo-key-123' http://localhost:8080/api/v1/tender${NC}"
        ;;
    2)
        echo -e "\n${GREEN}Starting in development mode...${NC}"
        go mod download
        go run cmd/server/main.go
        ;;
    3)
        echo -e "\n${GREEN}Building Docker image...${NC}"
        docker build -t go-data-gateway .
        echo -e "${GREEN}✅ Image built successfully${NC}"
        ;;
    4)
        echo -e "\n${YELLOW}Stopping all services...${NC}"
        docker-compose down
        echo -e "${GREEN}✅ Services stopped${NC}"
        ;;
    5)
        echo -e "\n${GREEN}Showing logs (Ctrl+C to exit)...${NC}"
        docker-compose logs -f go-gateway
        ;;
    6)
        echo -e "\n${GREEN}Running tests...${NC}"
        go test -v ./...
        ;;
    *)
        echo -e "${RED}Invalid option${NC}"
        ;;
esac