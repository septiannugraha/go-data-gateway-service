#!/bin/bash

# Docker Compose Startup Script for Go Data Gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Go Data Gateway - Docker Compose Startup${NC}"
echo "==========================================="

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file not found. Creating from .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✓ Created .env file. Please update it with your credentials.${NC}"
fi

# Parse command line arguments
MODE=${1:-production}

case "$MODE" in
    "dev"|"development")
        echo -e "${YELLOW}Starting in DEVELOPMENT mode...${NC}"
        docker-compose up --build
        ;;

    "prod"|"production")
        echo -e "${GREEN}Starting in PRODUCTION mode...${NC}"
        # Don't use override file in production
        docker-compose -f docker-compose.yml up -d --build
        echo -e "${GREEN}✓ Services started in background${NC}"
        echo ""
        echo "Services available at:"
        echo "  - API Gateway: http://localhost:8081"
        echo "  - Cache Stats: http://localhost:8081/cache/stats"
        echo "  - Prometheus:  http://localhost:9090"
        echo "  - Grafana:     http://localhost:3000 (admin/admin)"
        echo ""
        echo "View logs: docker-compose logs -f gateway"
        ;;

    "stop")
        echo -e "${YELLOW}Stopping all services...${NC}"
        docker-compose down
        echo -e "${GREEN}✓ All services stopped${NC}"
        ;;

    "clean")
        echo -e "${RED}Stopping and removing all containers, volumes, and networks...${NC}"
        docker-compose down -v
        echo -e "${GREEN}✓ Cleanup complete${NC}"
        ;;

    "logs")
        docker-compose logs -f gateway
        ;;

    "stats")
        echo "Cache Statistics:"
        curl -s http://localhost:8081/cache/stats | jq
        ;;

    *)
        echo "Usage: $0 [dev|prod|stop|clean|logs|stats]"
        echo ""
        echo "Commands:"
        echo "  dev    - Start in development mode with hot reload"
        echo "  prod   - Start in production mode (background)"
        echo "  stop   - Stop all services"
        echo "  clean  - Stop and remove all containers/volumes"
        echo "  logs   - Show gateway logs"
        echo "  stats  - Show cache statistics"
        exit 1
        ;;
esac