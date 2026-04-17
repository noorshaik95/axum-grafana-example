#!/bin/bash

# Slate - Install Dependencies Script
# This script installs all dependencies for all services in the project

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main installation
print_header "Slate - Installing Dependencies"

# Check for required tools
print_info "Checking for required tools..."

MISSING_TOOLS=()

if ! command_exists cargo; then
    MISSING_TOOLS+=("Rust/Cargo")
fi

if ! command_exists go; then
    MISSING_TOOLS+=("Go")
fi

if ! command_exists node; then
    MISSING_TOOLS+=("Node.js")
fi

if ! command_exists npm; then
    MISSING_TOOLS+=("npm")
fi

if ! command_exists docker; then
    MISSING_TOOLS+=("Docker")
fi

if ! command_exists docker-compose; then
    if ! docker compose version >/dev/null 2>&1; then
        MISSING_TOOLS+=("Docker Compose")
    fi
fi

if [ ${#MISSING_TOOLS[@]} -ne 0 ]; then
    print_error "Missing required tools:"
    for tool in "${MISSING_TOOLS[@]}"; do
        echo "  - $tool"
    done
    echo ""
    print_info "Please install the missing tools and try again."
    print_info "Visit the following for installation instructions:"
    echo "  - Rust: https://rustup.rs/"
    echo "  - Go: https://go.dev/doc/install"
    echo "  - Node.js: https://nodejs.org/"
    echo "  - Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

print_success "All required tools are installed"

# Install Rust services dependencies
print_header "Installing Rust Services Dependencies"

# API Gateway
print_info "Installing API Gateway dependencies..."
cd services/api-gateway
if cargo fetch; then
    print_success "API Gateway dependencies installed"
else
    print_error "Failed to install API Gateway dependencies"
    exit 1
fi
cd ../..

# Content Management Service
print_info "Installing Content Management Service dependencies..."
cd services/content-management-service
if cargo fetch; then
    print_success "Content Management Service dependencies installed"
else
    print_error "Failed to install Content Management Service dependencies"
    exit 1
fi
cd ../..

# Install Go services dependencies
print_header "Installing Go Services Dependencies"

# User Auth Service
print_info "Installing User Auth Service dependencies..."
cd services/user-auth-service
if go mod download; then
    print_success "User Auth Service dependencies installed"
else
    print_error "Failed to install User Auth Service dependencies"
    exit 1
fi
cd ../..

# Assignment Grading Service
print_info "Installing Assignment Grading Service dependencies..."
cd services/assignment-grading-service
if go mod download; then
    print_success "Assignment Grading Service dependencies installed"
else
    print_error "Failed to install Assignment Grading Service dependencies"
    exit 1
fi
cd ../..

# Install Node.js services dependencies
print_header "Installing Node.js Services Dependencies"

# Course Service
print_info "Installing Course Service dependencies..."
cd services/course-service
if npm install; then
    print_success "Course Service dependencies installed"
else
    print_error "Failed to install Course Service dependencies"
    exit 1
fi
cd ../..

# Frontend Student App
print_info "Installing Frontend Student App dependencies..."
cd frontend/student
if npm install; then
    print_success "Frontend Student App dependencies installed"
else
    print_error "Failed to install Frontend Student App dependencies"
    exit 1
fi
cd ../..

# Frontend Shared
print_info "Installing Frontend Shared dependencies..."
cd frontend/shared
if npm install; then
    print_success "Frontend Shared dependencies installed"
else
    print_error "Failed to install Frontend Shared dependencies"
    exit 1
fi
cd ../..

# Summary
print_header "Installation Complete"
print_success "All dependencies have been installed successfully!"
print_info "Next steps:"
echo "  1. Copy .env.example files to .env in each service directory"
echo "  2. Run './scripts/manage-services.sh start' to start all services"
echo "  3. Access Grafana at http://localhost:3000 (admin/admin)"
echo "  4. Access API Gateway at http://localhost:8080"
