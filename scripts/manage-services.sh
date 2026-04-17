#!/bin/bash

# Slate - Service Management Script
# This script manages Docker services (start, stop, restart, rebuild, logs)

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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
    echo -e "${CYAN}ℹ $1${NC}"
}

# Check if docker-compose or docker compose is available
get_docker_compose_cmd() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        print_error "Docker Compose is not installed"
        exit 1
    fi
}

DOCKER_COMPOSE=$(get_docker_compose_cmd)

# Usage information
usage() {
    echo "Usage: $0 {start|stop|restart|rebuild|logs|status|clean|dev} [service-name]"
    echo ""
    echo "Commands:"
    echo "  start       - Start all services (or specific service)"
    echo "  stop        - Stop all services (or specific service)"
    echo "  restart     - Restart all services (or specific service)"
    echo "  rebuild     - Rebuild and restart all services (or specific service)"
    echo "  logs        - Show logs for all services (or specific service)"
    echo "  status      - Show status of all services"
    echo "  clean       - Stop and remove all containers, networks, and volumes"
    echo "  dev         - Start only infrastructure services (DBs, observability)"
    echo ""
    echo "Service names:"
    echo "  api-gateway, user-auth-service, course-service,"
    echo "  assignment-grading-service, content-management-service,"
    echo "  postgres, redis, mongodb, minio, elasticsearch,"
    echo "  prometheus, tempo, loki, grafana, kafka"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # Start all services"
    echo "  $0 start api-gateway        # Start only API Gateway"
    echo "  $0 rebuild user-auth-service # Rebuild and restart User Auth Service"
    echo "  $0 logs api-gateway         # Show logs for API Gateway"
    echo "  $0 dev                      # Start only infrastructure"
    exit 1
}

# Start services
start_services() {
    local service=$1
    print_header "Starting Services"
    
    if [ -z "$service" ]; then
        print_info "Starting all services..."
        $DOCKER_COMPOSE up -d
        print_success "All services started"
    else
        print_info "Starting $service..."
        $DOCKER_COMPOSE up -d "$service"
        print_success "$service started"
    fi
    
    print_info "Waiting for services to be healthy..."
    sleep 5
    show_status
}

# Stop services
stop_services() {
    local service=$1
    print_header "Stopping Services"
    
    if [ -z "$service" ]; then
        print_info "Stopping all services..."
        $DOCKER_COMPOSE down
        print_success "All services stopped"
    else
        print_info "Stopping $service..."
        $DOCKER_COMPOSE stop "$service"
        print_success "$service stopped"
    fi
}

# Restart services
restart_services() {
    local service=$1
    print_header "Restarting Services"
    
    if [ -z "$service" ]; then
        print_info "Restarting all services..."
        $DOCKER_COMPOSE restart
        print_success "All services restarted"
    else
        print_info "Restarting $service..."
        $DOCKER_COMPOSE restart "$service"
        print_success "$service restarted"
    fi
    
    print_info "Waiting for services to be healthy..."
    sleep 5
    show_status
}

# Rebuild services
rebuild_services() {
    local service=$1
    print_header "Rebuilding Services"
    
    if [ -z "$service" ]; then
        print_info "Rebuilding all services..."
        $DOCKER_COMPOSE build --no-cache
        print_success "All services rebuilt"
        print_info "Starting services..."
        $DOCKER_COMPOSE up -d
        print_success "All services started"
    else
        print_info "Rebuilding $service..."
        $DOCKER_COMPOSE build --no-cache "$service"
        print_success "$service rebuilt"
        print_info "Starting $service..."
        $DOCKER_COMPOSE up -d "$service"
        print_success "$service started"
    fi
    
    print_info "Waiting for services to be healthy..."
    sleep 5
    show_status
}

# Show logs
show_logs() {
    local service=$1
    print_header "Service Logs"
    
    if [ -z "$service" ]; then
        print_info "Showing logs for all services (Ctrl+C to exit)..."
        $DOCKER_COMPOSE logs -f
    else
        print_info "Showing logs for $service (Ctrl+C to exit)..."
        $DOCKER_COMPOSE logs -f "$service"
    fi
}

# Show status
show_status() {
    print_header "Service Status"
    $DOCKER_COMPOSE ps
    
    echo ""
    print_info "Service URLs:"
    echo "  API Gateway:        http://localhost:8080"
    echo "  Grafana:            http://localhost:3000 (admin/admin)"
    echo "  Prometheus:         http://localhost:9090"
    echo "  MinIO Console:      http://localhost:9001 (minioadmin/minioadmin)"
    echo "  Tempo:              http://localhost:3200"
    echo "  Loki:               http://localhost:3100"
    echo ""
    print_info "Database Ports:"
    echo "  PostgreSQL (Auth):  localhost:5432"
    echo "  PostgreSQL (CMS):   localhost:5433"
    echo "  MongoDB:            localhost:27017"
    echo "  Redis:              localhost:6379"
    echo "  Elasticsearch:      localhost:9200"
    echo "  Kafka:              localhost:9092"
}

# Clean everything
clean_services() {
    print_header "Cleaning Services"
    print_warning "This will remove all containers, networks, and volumes!"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Stopping and removing all services..."
        $DOCKER_COMPOSE down -v --remove-orphans
        print_success "All services cleaned"
        
        print_info "Removing dangling images..."
        docker image prune -f
        print_success "Cleanup complete"
    else
        print_info "Cleanup cancelled"
    fi
}

# Start only infrastructure (dev mode)
start_dev() {
    print_header "Starting Development Infrastructure"
    print_info "Starting databases and observability stack..."
    
    $DOCKER_COMPOSE -f docker-compose.dev.yml up -d
    
    print_success "Development infrastructure started"
    print_info "You can now run services locally"
    echo ""
    print_info "Available services:"
    echo "  PostgreSQL:  localhost:5432"
    echo "  Prometheus:  http://localhost:9090"
    echo "  Tempo:       http://localhost:3200"
    echo "  Loki:        http://localhost:3100"
    echo "  Grafana:     http://localhost:3000 (admin/admin)"
}

# Main script
if [ $# -eq 0 ]; then
    usage
fi

COMMAND=$1
SERVICE=${2:-}

case $COMMAND in
    start)
        start_services "$SERVICE"
        ;;
    stop)
        stop_services "$SERVICE"
        ;;
    restart)
        restart_services "$SERVICE"
        ;;
    rebuild)
        rebuild_services "$SERVICE"
        ;;
    logs)
        show_logs "$SERVICE"
        ;;
    status)
        show_status
        ;;
    clean)
        clean_services
        ;;
    dev)
        start_dev
        ;;
    *)
        print_error "Unknown command: $COMMAND"
        usage
        ;;
esac
