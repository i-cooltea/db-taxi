#!/bin/bash

# DB-Taxi Production Deployment Script
# This script deploys DB-Taxi in production mode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="db-taxi"
IMAGE_TAG="${IMAGE_TAG:-latest}"
COMPOSE_FILE="docker-compose.prod.yml"
BACKUP_DIR="./backups"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_requirements() {
    log_step "Checking requirements..."
    
    local missing_requirements=0
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        missing_requirements=1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        missing_requirements=1
    fi
    
    if [ $missing_requirements -eq 1 ]; then
        log_error "Please install missing requirements"
        exit 1
    fi
    
    log_info "Requirements check passed ✓"
}

check_environment() {
    log_step "Checking environment configuration..."
    
    if [ ! -f .env ]; then
        log_error ".env file not found"
        log_info "Please create .env file from .env.example and configure it"
        exit 1
    fi
    
    # Source .env file
    source .env
    
    # Check required variables
    local required_vars=("DB_HOST" "DB_USERNAME" "DB_PASSWORD" "DB_DATABASE")
    local missing_vars=0
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            log_error "Required environment variable $var is not set"
            missing_vars=1
        fi
    done
    
    if [ $missing_vars -eq 1 ]; then
        log_error "Please configure all required environment variables in .env"
        exit 1
    fi
    
    log_info "Environment configuration check passed ✓"
}

backup_config() {
    log_step "Backing up current configuration..."
    
    mkdir -p "${BACKUP_DIR}"
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/config_backup_${timestamp}.tar.gz"
    
    tar -czf "${backup_file}" .env configs/ 2>/dev/null || true
    
    if [ -f "${backup_file}" ]; then
        log_info "Configuration backed up to: ${backup_file}"
    else
        log_warn "Failed to create backup"
    fi
}

build_image() {
    log_step "Building production Docker image..."
    
    docker build \
        --tag "${IMAGE_NAME}:${IMAGE_TAG}" \
        --tag "${IMAGE_NAME}:production" \
        --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
        --build-arg VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
        .
    
    if [ $? -eq 0 ]; then
        log_info "Docker image built successfully ✓"
    else
        log_error "Failed to build Docker image"
        exit 1
    fi
}

run_migrations() {
    log_step "Running database migrations..."
    
    # Check if migration command exists
    if [ -f "./scripts/migrate.sh" ]; then
        source .env
        ./scripts/migrate.sh -h "${DB_HOST}" -u "${DB_USERNAME}" -P "${DB_PASSWORD}" -d "${DB_DATABASE}"
        
        if [ $? -eq 0 ]; then
            log_info "Database migrations completed ✓"
        else
            log_error "Database migrations failed"
            exit 1
        fi
    else
        log_warn "Migration script not found, skipping migrations"
    fi
}

deploy() {
    log_step "Deploying to production..."
    
    # Pull latest images (if using registry)
    # docker-compose -f "${COMPOSE_FILE}" pull
    
    # Stop existing containers gracefully
    log_info "Stopping existing containers..."
    docker-compose -f "${COMPOSE_FILE}" down --timeout 30
    
    # Start new containers
    log_info "Starting new containers..."
    docker-compose -f "${COMPOSE_FILE}" up -d
    
    if [ $? -eq 0 ]; then
        log_info "Containers started successfully ✓"
    else
        log_error "Failed to start containers"
        exit 1
    fi
}

wait_for_health() {
    log_step "Waiting for application to be healthy..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:${APP_PORT:-8080}/health &> /dev/null; then
            log_info "Application is healthy ✓"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    echo ""
    log_error "Application failed to become healthy"
    log_info "Check logs with: docker-compose -f ${COMPOSE_FILE} logs"
    return 1
}

show_status() {
    log_step "Deployment status:"
    
    echo ""
    docker-compose -f "${COMPOSE_FILE}" ps
    
    echo ""
    log_info "Application URL: http://localhost:${APP_PORT:-8080}"
    log_info "Health endpoint: http://localhost:${APP_PORT:-8080}/health"
}

cleanup_old_images() {
    log_step "Cleaning up old Docker images..."
    
    docker image prune -f
    
    log_info "Cleanup completed ✓"
}

# Main script
main() {
    echo ""
    log_info "╔════════════════════════════════════════╗"
    log_info "║  DB-Taxi Production Deployment         ║"
    log_info "╚════════════════════════════════════════╝"
    echo ""
    
    # Confirmation prompt
    log_warn "This will deploy DB-Taxi to production"
    read -p "Are you sure you want to continue? (yes/no) " -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi
    
    # Run deployment steps
    check_requirements
    check_environment
    backup_config
    build_image
    run_migrations
    deploy
    
    if wait_for_health; then
        show_status
        cleanup_old_images
        
        echo ""
        log_info "╔════════════════════════════════════════╗"
        log_info "║  Deployment Successful! ✓              ║"
        log_info "╚════════════════════════════════════════╝"
        echo ""
        
        log_info "Useful commands:"
        log_info "  View logs:    docker-compose -f ${COMPOSE_FILE} logs -f"
        log_info "  Stop:         docker-compose -f ${COMPOSE_FILE} down"
        log_info "  Restart:      docker-compose -f ${COMPOSE_FILE} restart"
        log_info "  Status:       docker-compose -f ${COMPOSE_FILE} ps"
        log_info "  Shell:        docker-compose -f ${COMPOSE_FILE} exec db-taxi sh"
        
        echo ""
        read -p "Do you want to view logs? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose -f "${COMPOSE_FILE}" logs -f
        fi
    else
        log_error "Deployment failed"
        log_info "Rolling back..."
        docker-compose -f "${COMPOSE_FILE}" down
        exit 1
    fi
}

# Run main function
main
