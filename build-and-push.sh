#!/bin/bash

# Docker build, tag and push script for DB-Taxi

set -e

# Default values
DOCKER_HUB_USER=""
IMAGE_NAME="db-taxi"
TAG="latest"
DOCKERFILE="Dockerfile"
BUILD_CONTEXT="."

# Usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build, tag and push DB-Taxi Docker image to Docker Hub"
    echo ""
    echo "Options:"
    echo "  -u, --user USER      Docker Hub username (required)"
    echo "  -n, --name NAME      Docker image name (default: $IMAGE_NAME)"
    echo "  -t, --tag TAG        Docker image tag (default: $TAG)"
    echo "  -f, --file FILE      Dockerfile path (default: $DOCKERFILE)"
    echo "  -c, --context PATH   Build context (default: $BUILD_CONTEXT)"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Example:"
    echo "  $0 -u myusername -t v1.0.0"
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--user)
            DOCKER_HUB_USER="$2"
            shift 2
            ;;
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -f|--file)
            DOCKERFILE="$2"
            shift 2
            ;;
        -c|--context)
            BUILD_CONTEXT="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Error: Unknown option $1"
            usage
            ;;
    esac
done

# Check if Docker Hub username is provided
if [[ -z "$DOCKER_HUB_USER" ]]; then
    echo "Error: Docker Hub username is required"
    usage
fi

# Build the image
echo "Building Docker image..."
docker build -f "$DOCKERFILE" -t "$DOCKER_HUB_USER/$IMAGE_NAME:$TAG" "$BUILD_CONTEXT"

# Tag as latest if not already
echo "Tagging image as latest..."
docker tag "$DOCKER_HUB_USER/$IMAGE_NAME:$TAG" "$DOCKER_HUB_USER/$IMAGE_NAME:latest"

# Push to Docker Hub
echo "Pushing images to Docker Hub..."
docker push "$DOCKER_HUB_USER/$IMAGE_NAME:$TAG"
docker push "$DOCKER_HUB_USER/$IMAGE_NAME:latest"

echo ""
echo "✅ Docker image build and push completed successfully!"
echo ""
echo "Image tags pushed:"
echo "  - $DOCKER_HUB_USER/$IMAGE_NAME:$TAG"
echo "  - $DOCKER_HUB_USER/$IMAGE_NAME:latest"