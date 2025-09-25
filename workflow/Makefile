# Makefile for n8n User Onboarding Workflow Testing
# Comprehensive test automation for all testing levels

.PHONY: help test-all test-unit test-integration test-e2e test-load test-security \
        test-smoke test-ci coverage clean setup-test start-test stop-test \
        test-report lint format install-deps

# Color output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
help:
	@echo "$(BLUE)n8n Workflow Test Automation$(NC)"
	@echo "=============================="
	@echo ""
	@echo "$(YELLOW)Setup Commands:$(NC)"
	@echo "  make setup-test      - Set up test environment"
	@echo "  make install-deps    - Install test dependencies"
	@echo ""
	@echo "$(YELLOW)Test Commands:$(NC)"
	@echo "  make test-all        - Run all tests"
	@echo "  make test-smoke      - Quick smoke tests (5 min)"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration- Run integration tests"
	@echo "  make test-e2e        - Run end-to-end tests"
	@echo "  make test-load       - Run load tests"
	@echo "  make test-security   - Run security tests"
	@echo "  make test-ci         - CI/CD test suite"
	@echo ""
	@echo "$(YELLOW)Reporting:$(NC)"
	@echo "  make coverage        - Generate coverage report"
	@echo "  make test-report     - Generate HTML test report"
	@echo ""
	@echo "$(YELLOW)Utilities:$(NC)"
	@echo "  make lint            - Run linters"
	@echo "  make format          - Format code"
	@echo "  make clean           - Clean test artifacts"
	@echo "  make start-test      - Start test environment"
	@echo "  make stop-test       - Stop test environment"

# Environment setup
setup-test:
	@echo "$(YELLOW)Setting up test environment...$(NC)"
	@cp .env.example .env.test 2>/dev/null || true
	@mkdir -p test-results tests/reports
	@docker-compose -f docker-compose.test.yml build
	@echo "$(GREEN)✓ Test environment ready$(NC)"

install-deps:
	@echo "$(YELLOW)Installing test dependencies...$(NC)"
	@npm install --save-dev jest @types/jest supertest
	@pip install -r requirements-test.txt
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

# Test environment management
start-test:
	@echo "$(YELLOW)Starting test environment...$(NC)"
	@docker-compose -f docker-compose.test.yml up -d
	@sleep 10  # Wait for services
	@echo "$(GREEN)✓ Test environment running$(NC)"

stop-test:
	@echo "$(YELLOW)Stopping test environment...$(NC)"
	@docker-compose -f docker-compose.test.yml down
	@echo "$(GREEN)✓ Test environment stopped$(NC)"

# Testing targets
test-all: start-test test-unit test-integration test-e2e test-security stop-test
	@echo "$(GREEN)✓ All tests completed$(NC)"
	@make test-report

test-smoke: start-test
	@echo "$(YELLOW)Running smoke tests...$(NC)"
	@./scripts/test.sh
	@echo "$(GREEN)✓ Smoke tests passed$(NC)"
	@make stop-test

test-unit:
	@echo "$(YELLOW)Running unit tests...$(NC)"
	@echo "Testing password generator..."
	@npx jest tests/unit/test_password_generator.js --coverage
	@echo "Testing Fusio client..."
	@npx jest tests/unit/test_fusio_client.js --coverage
	@echo "$(GREEN)✓ Unit tests completed$(NC)"

test-integration: start-test
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@docker-compose -f docker-compose.test.yml run --rm test-runner \
		pytest tests/integration/ -v --tb=short
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

test-e2e: start-test
	@echo "$(YELLOW)Running end-to-end tests...$(NC)"
	@docker-compose -f docker-compose.test.yml run --rm test-runner \
		pytest tests/e2e/ -v --tb=short
	@echo "$(GREEN)✓ E2E tests completed$(NC)"

test-load:
	@echo "$(YELLOW)Running load tests...$(NC)"
	@echo "Starting with 10 concurrent users..."
	@docker run --rm -v $(PWD)/tests/load:/scripts \
		--network workflow_test-network \
		loadimpact/k6 run /scripts/k6_script.js
	@echo ""
	@echo "Running Locust load test..."
	@locust -f tests/load/locustfile.py \
		--host http://localhost:5679 \
		--users 50 --spawn-rate 5 \
		--run-time 60s --headless
	@echo "$(GREEN)✓ Load tests completed$(NC)"

test-security:
	@echo "$(YELLOW)Running security tests...$(NC)"
	@echo "Checking for hardcoded secrets..."
	@grep -r "password\|secret\|api_key" services/ --exclude="*.test.js" || true
	@echo "Testing SQL injection attempts..."
	@python tests/security/test_injections.py
	@echo "Testing rate limiting..."
	@python tests/security/test_rate_limit.py
	@echo "$(GREEN)✓ Security tests completed$(NC)"

# CI/CD simulation
test-ci: lint
	@echo "$(BLUE)Running CI/CD test suite...$(NC)"
	@make test-unit
	@make start-test
	@make test-integration
	@make test-smoke
	@make stop-test
	@make coverage
	@echo "$(GREEN)✓ CI/CD tests passed$(NC)"

# Coverage and reporting
coverage:
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	@mkdir -p test-results/coverage
	@npx jest --coverage --coverageDirectory=test-results/coverage/js
	@pytest tests/ --cov=services --cov-report=html:test-results/coverage/python
	@echo "$(GREEN)✓ Coverage report generated$(NC)"
	@echo "View at: test-results/coverage/index.html"

test-report:
	@echo "$(YELLOW)Generating test report...$(NC)"
	@docker-compose -f docker-compose.test.yml run --rm test-runner \
		pytest tests/ --html=test-results/report.html --self-contained-html
	@echo "$(GREEN)✓ Test report generated$(NC)"
	@echo "View at: test-results/report.html"

# Code quality
lint:
	@echo "$(YELLOW)Running linters...$(NC)"
	@npx eslint services/*.js tests/unit/*.js --fix || true
	@python -m flake8 tests/ --max-line-length=120
	@echo "$(GREEN)✓ Linting completed$(NC)"

format:
	@echo "$(YELLOW)Formatting code...$(NC)"
	@npx prettier --write "services/*.js" "tests/**/*.js"
	@python -m black tests/
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Cleanup
clean:
	@echo "$(YELLOW)Cleaning test artifacts...$(NC)"
	@rm -rf test-results/* coverage/ .coverage .pytest_cache
	@rm -f *.log test-*.json
	@docker-compose -f docker-compose.test.yml down -v
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

# Watch mode for development
watch:
	@echo "$(YELLOW)Starting test watch mode...$(NC)"
	@npx jest --watch

# Performance benchmarks
benchmark:
	@echo "$(YELLOW)Running performance benchmarks...$(NC)"
	@python tests/benchmarks/benchmark_password.py
	@node tests/benchmarks/benchmark_api.js
	@echo "$(GREEN)✓ Benchmarks completed$(NC)"

# Specific test targets with parameters
test-specific:
	@echo "$(YELLOW)Running specific test: $(TEST)$(NC)"
	@pytest tests/ -k $(TEST) -v

test-verbose:
	@make test-all VERBOSE=true

# Docker utilities
logs:
	@docker-compose -f docker-compose.test.yml logs -f

ps:
	@docker-compose -f docker-compose.test.yml ps

# Installation verification
verify:
	@echo "$(YELLOW)Verifying test setup...$(NC)"
	@command -v docker >/dev/null 2>&1 && echo "$(GREEN)✓ Docker installed$(NC)" || echo "$(RED)✗ Docker not found$(NC)"
	@command -v python3 >/dev/null 2>&1 && echo "$(GREEN)✓ Python installed$(NC)" || echo "$(RED)✗ Python not found$(NC)"
	@command -v node >/dev/null 2>&1 && echo "$(GREEN)✓ Node.js installed$(NC)" || echo "$(RED)✗ Node.js not found$(NC)"
	@command -v npm >/dev/null 2>&1 && echo "$(GREEN)✓ npm installed$(NC)" || echo "$(RED)✗ npm not found$(NC)"
	@test -f docker-compose.test.yml && echo "$(GREEN)✓ Test compose file exists$(NC)" || echo "$(RED)✗ Test compose file missing$(NC)"

# Quick test for development
quick:
	@make test-smoke

# Full CI pipeline simulation
pipeline: clean setup-test test-ci test-report
	@echo "$(GREEN)════════════════════════════════════$(NC)"
	@echo "$(GREEN)  Pipeline completed successfully!  $(NC)"
	@echo "$(GREEN)════════════════════════════════════$(NC)"