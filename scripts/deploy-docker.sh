#!/bin/bash

# DB-Taxi Docker Deployment Script
# This script builds and deploys DB-Taxi using Docker

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="db-taxi"
IMAGE_TAG="${IMAGE_TAG:-latest}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"

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

check_requirements() {
    log_info "Checking requirements..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    log_info "Requirements check passed"
}

build_image() {
    log_info "Building Docker image: ${IMAGE_NAME}:${IMAGE_TAG}"
    docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .
    
    if [ $? -eq 0 ]; then
        log_info "Docker image built successfully"
    else
        log_error "Failed to build Docker image"
        exit 1
    fi
}

deploy() {
    log_info "Deploying with Docker Compose: ${COMPOSE_FILE}"
    
    # Check if .env file exists
    if [ ! -f .env ]; then
        log_warn ".env file not found. Creating from .env.example..."
        if [ -f .env.example ]; then
            cp .env.example .env
            log_warn "Please update .env file with your configuration"
            exit 1
        else
            log_error ".env.example not found"
            exit 1
        fi
    fi
    
    # Stop existing containers
    log_info "Stopping existing containers..."
    docker-compose -f "${COMPOSE_FILE}" down
    
    # Start new containers
    log_info "Starting containers..."
    docker-compose -f "${COMPOSE_FILE}" up -d
    
    if [ $? -eq 0 ]; then
        log_info "Deployment successful"
    else
        log_error "Deployment failed"
        exit 1
    fi
}

show_status() {
    log_info "Container status:"
    docker-compose -f "${COMPOSE_FILE}" ps
    
    log_info ""
    log_info "Checking health..."
    sleep 5
    
    if curl -f http://localhost:8080/health &> /dev/null; then
        log_info "Health check passed ✓"
    else
        log_warn "Health check failed. Check logs with: docker-compose logs db-taxi"
    fi
}

show_logs() {
    log_info "Showing logs (Ctrl+C to exit)..."
    docker-compose -f "${COMPOSE_FILE}" logs -f
}

# Main script
main() {
    log_info "DB-Taxi Docker Deployment"
    log_info "========================="
    
    check_requirements
    build_image
    deploy
    show_status
    
    log_info ""
    log_info "Deployment complete!"
    log_info "Access DB-Taxi at: http://localhost:8080"
    log_info ""
    log_info "Useful commands:"
    log_info "  View logs:    docker-compose -f ${COMPOSE_FILE} logs -f"
    log_info "  Stop:         docker-compose -f ${COMPOSE_FILE} down"
    log_info "  Restart:      docker-compose -f ${COMPOSE_FILE} restart"
    log_info "  Status:       docker-compose -f ${COMPOSE_FILE} ps"
    
    # Ask if user wants to see logs
    read -p "Do you want to view logs? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        show_logs
    fi
}

# Run main function
main
