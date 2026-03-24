#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "========================================"
echo "  MonkeyCode Integration Tests"
echo "========================================"
echo ""

TASKFLOW_RUNNING=false
RUNNER_RUNNING=false
REDIS_RUNNING=false
DOCKER_RUNNING=false

echo "Checking services..."

if redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Redis is running"
    REDIS_RUNNING=true
else
    echo -e "${RED}✗${NC} Redis is not running"
fi

if docker info > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Docker is running"
    DOCKER_RUNNING=true
else
    echo -e "${RED}✗${NC} Docker is not running"
fi

if lsof -i :8889 | grep -q LISTEN; then
    echo -e "${GREEN}✓${NC} TaskFlow HTTP is running on :8889"
    TASKFLOW_RUNNING=true
else
    echo -e "${RED}✗${NC} TaskFlow HTTP is not running"
fi

if lsof -i :8080 | grep -q LISTEN; then
    echo -e "${GREEN}✓${NC} Runner HTTP is running on :8080"
    RUNNER_RUNNING=true
else
    echo -e "${RED}✗${NC} Runner HTTP is not running"
fi

echo ""
echo "========================================"
echo "  Running Unit Tests"
echo "========================================"
echo ""

echo "Running TaskFlow tests..."
cd /Users/wanglx/dennis/project/MonkeyCode/taskflow
if make test; then
    echo -e "${GREEN}✓${NC} TaskFlow tests passed"
else
    echo -e "${RED}✗${NC} TaskFlow tests failed"
    exit 1
fi

echo ""
echo "Running Runner tests..."
cd /Users/wanglx/dennis/project/MonkeyCode/runner
if make test; then
    echo -e "${GREEN}✓${NC} Runner tests passed"
else
    echo -e "${RED}✗${NC} Runner tests failed"
    exit 1
fi

if [ "$TASKFLOW_RUNNING" = true ]; then
    echo ""
    echo "========================================"
    echo "  Testing TaskFlow API Endpoints"
    echo "========================================"
    echo ""

    echo "Testing /internal/stats..."
    response=$(curl -s http://localhost:8889/internal/stats)
    if echo "$response" | grep -q '"code":0'; then
        echo -e "${GREEN}✓${NC} Stats endpoint working"
    else
        echo -e "${RED}✗${NC} Stats endpoint failed"
        echo "Response: $response"
    fi

    echo ""
    echo "Testing /internal/host/is-online..."
    response=$(curl -s -X POST http://localhost:8889/internal/host/is-online \
        -H "Content-Type: application/json" \
        -d '{"runner_ids": []}')
    if echo "$response" | grep -q '"code":0'; then
        echo -e "${GREEN}✓${NC} Host check endpoint working"
    else
        echo -e "${RED}✗${NC} Host check endpoint failed"
        echo "Response: $response"
    fi

    echo ""
    echo "Testing /internal/vm/is-online..."
    response=$(curl -s -X POST http://localhost:8889/internal/vm/is-online \
        -H "Content-Type: application/json" \
        -d '{"vm_ids": []}')
    if echo "$response" | grep -q '"code":0'; then
        echo -e "${GREEN}✓${NC} VM check endpoint working"
    else
        echo -e "${RED}✗${NC} VM check endpoint failed"
        echo "Response: $response"
    fi
fi

if [ "$RUNNER_RUNNING" = true ]; then
    echo ""
    echo "========================================"
    echo "  Testing Runner API Endpoints"
    echo "========================================"
    echo ""

    echo "Testing /health..."
    response=$(curl -s http://localhost:8080/health)
    if [ "$response" = "ok" ]; then
        echo -e "${GREEN}✓${NC} Health endpoint working"
    else
        echo -e "${RED}✗${NC} Health endpoint failed"
        echo "Response: $response"
    fi
fi

if [ "$REDIS_RUNNING" = true ]; then
    echo ""
    echo "========================================"
    echo "  Testing Redis Connection"
    echo "========================================"
    echo ""

    echo "Testing Redis connection..."
    if redis-cli ping | grep -q PONG; then
        echo -e "${GREEN}✓${NC} Redis connection working"
    else
        echo -e "${RED}✗${NC} Redis connection failed"
    fi
fi

if [ "$DOCKER_RUNNING" = true ]; then
    echo ""
    echo "========================================"
    echo "  Testing Docker Integration"
    echo "========================================"
    echo ""

    echo "Testing Docker connection..."
    if docker ps > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Docker connection working"
        container_count=$(docker ps -q | wc -l)
        echo "  Running containers: $container_count"
    else
        echo -e "${RED}✗${NC} Docker connection failed"
    fi
fi

echo ""
echo "========================================"
echo "  Integration Tests Summary"
echo "========================================"
echo ""

if [ "$TASKFLOW_RUNNING" = true ]; then
    echo -e "${GREEN}✓${NC} TaskFlow: Running"
else
    echo -e "${RED}✗${NC} TaskFlow: Not running"
fi

if [ "$RUNNER_RUNNING" = true ]; then
    echo -e "${GREEN}✓${NC} Runner: Running"
else
    echo -e "${RED}✗${NC} Runner: Not running"
fi

if [ "$REDIS_RUNNING" = true ]; then
    echo -e "${GREEN}✓${NC} Redis: Running"
else
    echo -e "${RED}✗${NC} Redis: Not running"
fi

if [ "$DOCKER_RUNNING" = true ]; then
    echo -e "${GREEN}✓${NC} Docker: Running"
else
    echo -e "${RED}✗${NC} Docker: Not running"
fi

echo ""
echo -e "${GREEN}All integration tests completed!${NC}"
echo ""
