#!/bin/bash

# End-to-End Integration Test Script for Maestro MCP Server
# Tests all major functionality including vector database operations

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Server configuration
HOST=${MAESTRO_MCP_SERVER_HOST:-localhost}
PORT=${MAESTRO_MCP_SERVER_PORT:-8030}
SERVER_URL="http://$HOST:$PORT"

# Function to print test header
print_test_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}🧪 E2E Integration Test Suite${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "${CYAN}📝 Testing Maestro MCP Server${NC}"
    echo -e "${CYAN}🔗 Server URL: $SERVER_URL${NC}"
}

# Function to print test section
print_section() {
    echo -e "\n${CYAN}📋 $1${NC}"
    echo -e "${CYAN}$(printf '=%.0s' {1..50})${NC}"
}

# Function to run a test
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_exit_code="${3:-0}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${YELLOW}Test $TOTAL_TESTS: $test_name${NC}"
    echo -e "${YELLOW}Command: $command${NC}"
    
    # Capture both stdout and stderr
    local output
    local exit_code
    
    if output=$(eval "$command" 2>&1); then
        exit_code=$?
        if [ $exit_code -eq "$expected_exit_code" ]; then
            echo -e "${GREEN}✅ PASSED${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}❌ FAILED (wrong exit code: $exit_code, expected: $expected_exit_code)${NC}"
            echo -e "${RED}Output: $output${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        exit_code=$?
        if [ $exit_code -eq "$expected_exit_code" ]; then
            echo -e "${GREEN}✅ PASSED${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}❌ FAILED (exit code: $exit_code)${NC}"
            echo -e "${RED}Output: $output${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
}

# Function to check server health
check_server_health() {
    echo -e "${YELLOW}🔍 Checking server health...${NC}"
    
    if curl -s "$SERVER_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Server is responding to health checks${NC}"
        return 0
    else
        echo -e "${RED}❌ Server is not responding to health checks${NC}"
        return 1
    fi
}

# Function to make HTTP request
# shellcheck disable=SC2317
make_request() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -n "$data" ]; then
        curl -s -X "$method" \
             -H "Content-Type: application/json" \
             -d "$data" \
             "$SERVER_URL$endpoint"
    else
        curl -s -X "$method" "$SERVER_URL$endpoint"
    fi
}

# Function to print final results
print_results() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}📊 E2E Test Results Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}✅ Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}❌ Failed: $FAILED_TESTS${NC}"
    echo -e "${CYAN}📈 Total:  $TOTAL_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed! E2E integration test successful!${NC}"
        exit 0
    else
        echo -e "\n${RED}💥 Some tests failed. Please check the output above.${NC}"
        exit 1
    fi
}

# Main execution
main() {
    print_test_header
    
    # Step 1: Lint and Build
    print_section "Step 1: Code Quality & Build"
    run_test "Lint code" "./lint.sh"
    run_test "Build binary" "./build.sh"
    
    # Step 2: Start Server
    print_section "Step 2: Start Server"
    run_test "Start server in daemon mode" "./start.sh --daemon"
    
    # Wait for server to start
    echo -e "${YELLOW}⏳ Waiting for server to start...${NC}"
    sleep 5
    
    # Step 3: Health Check
    print_section "Step 3: Health Check"
    if check_server_health; then
        run_test "Health check endpoint" "curl -s $SERVER_URL/health"
    else
        echo -e "${RED}❌ Server health check failed - skipping remaining tests${NC}"
        print_results
    fi
    
    # Step 4: MCP Tools Tests
    print_section "Step 4: MCP Tools Tests"
    
    # List available tools
    run_test "List available tools" "make_request GET /mcp/tools/list"
    
    # Create vector database
    run_test "Create Milvus vector database" "make_request POST /mcp/tools/call '{\"name\": \"create_vector_database\", \"arguments\": {\"db_name\": \"test_db\", \"db_type\": \"milvus\", \"collection_name\": \"test_collection\"}}'"
    
    # Setup database
    run_test "Setup vector database" "make_request POST /mcp/tools/call '{\"name\": \"setup_database\", \"arguments\": {\"db_name\": \"test_db\", \"embedding\": \"default\"}}'"
    
    # List databases
    run_test "List vector databases" "make_request POST /mcp/tools/call '{\"name\": \"list_databases\", \"arguments\": {}}'"
    
    # Step 5: Document Operations Tests
    print_section "Step 5: Document Operations Tests"
    
    # Write a document
    run_test "Write document" "make_request POST /mcp/tools/call '{\"name\": \"write_document\", \"arguments\": {\"db_name\": \"test_db\", \"url\": \"https://example.com/doc1\", \"text\": \"This is a test document about machine learning and artificial intelligence.\", \"metadata\": {\"author\": \"Test User\", \"category\": \"AI\"}}}'"
    
    # Write another document
    run_test "Write second document" "make_request POST /mcp/tools/call '{\"name\": \"write_document\", \"arguments\": {\"db_name\": \"test_db\", \"url\": \"https://example.com/doc2\", \"text\": \"This document discusses natural language processing and deep learning techniques.\", \"metadata\": {\"author\": \"Test User\", \"category\": \"NLP\"}}}'"
    
    # Count documents
    run_test "Count documents" "make_request POST /mcp/tools/call '{\"name\": \"count_documents\", \"arguments\": {\"db_name\": \"test_db\"}}'"
    
    # List documents
    run_test "List documents" "make_request POST /mcp/tools/call '{\"name\": \"list_documents\", \"arguments\": {\"db_name\": \"test_db\", \"limit\": 10, \"offset\": 0}}'"
    
    # Step 6: Query Tests
    print_section "Step 6: Query Tests"
    
    # Query documents
    run_test "Query documents" "make_request POST /mcp/tools/call '{\"name\": \"query\", \"arguments\": {\"db_name\": \"test_db\", \"query\": \"What is machine learning?\", \"limit\": 5}}'"
    
    # Query with different terms
    run_test "Query with different terms" "make_request POST /mcp/tools/call '{\"name\": \"query\", \"arguments\": {\"db_name\": \"test_db\", \"query\": \"natural language processing\", \"limit\": 3}}'"
    
    # Step 7: Error Handling Tests
    print_section "Step 7: Error Handling Tests"
    
    # Test with non-existent database
    run_test "Query non-existent database (should fail)" "make_request POST /mcp/tools/call '{\"name\": \"query\", \"arguments\": {\"db_name\": \"non_existent_db\", \"query\": \"test query\", \"limit\": 5}}'" 0
    
    # Test with invalid tool name
    run_test "Call non-existent tool (should fail)" "make_request POST /mcp/tools/call '{\"name\": \"non_existent_tool\", \"arguments\": {}}'" 0
    
    # Step 8: Cleanup Tests
    print_section "Step 8: Cleanup Tests"
    
    # Delete a document (if we can get document IDs)
    run_test "Delete document" "make_request POST /mcp/tools/call '{\"name\": \"delete_document\", \"arguments\": {\"db_name\": \"test_db\", \"document_id\": \"test_doc_1\"}}'"
    
    # Cleanup database
    run_test "Cleanup database" "make_request POST /mcp/tools/call '{\"name\": \"cleanup\", \"arguments\": {\"db_name\": \"test_db\"}}'"
    
    # Step 9: Stop Server
    print_section "Step 9: Stop Server"
    run_test "Stop server" "./stop.sh"
    
    # Print results
    print_results
}

# Run main function
main "$@"