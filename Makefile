.PHONY: help build run test migrate migrate-status migrate-version clean

# Default target
help:
	@echo "DB-Taxi Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build            Build the application"
	@echo "  run              Run the application"
	@echo "  test             Run tests"
	@echo "  migrate          Run database migrations"
	@echo "  migrate-status   Show migration status"
	@echo "  migrate-version  Show current migration version"
	@echo "  clean            Clean build artifacts"
	@echo ""
	@echo "Migration examples:"
	@echo "  make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb"
	@echo "  make migrate-status CONFIG=config.yaml"

# Build the application
build:
	@echo "Building DB-Taxi..."
	go build -o db-taxi main.go
	@echo "Build complete: ./db-taxi"

# Run the application
run:
	@echo "Running DB-Taxi..."
	go run main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run database migrations
migrate:
	@echo "Running database migrations..."
	@if [ -n "$(CONFIG)" ]; then \
		./scripts/migrate.sh migrate -c $(CONFIG); \
	else \
		./scripts/migrate.sh migrate \
			$(if $(HOST),-h $(HOST),) \
			$(if $(PORT),-p $(PORT),) \
			$(if $(USER),-u $(USER),) \
			$(if $(PASSWORD),-P $(PASSWORD),) \
			$(if $(DB),-d $(DB),); \
	fi

# Show migration status
migrate-status:
	@echo "Checking migration status..."
	@if [ -n "$(CONFIG)" ]; then \
		./scripts/migrate.sh status -c $(CONFIG); \
	else \
		./scripts/migrate.sh status \
			$(if $(HOST),-h $(HOST),) \
			$(if $(PORT),-p $(PORT),) \
			$(if $(USER),-u $(USER),) \
			$(if $(PASSWORD),-P $(PASSWORD),) \
			$(if $(DB),-d $(DB),); \
	fi

# Show current migration version
migrate-version:
	@echo "Checking migration version..."
	@if [ -n "$(CONFIG)" ]; then \
		./scripts/migrate.sh version -c $(CONFIG); \
	else \
		./scripts/migrate.sh version \
			$(if $(HOST),-h $(HOST),) \
			$(if $(PORT),-p $(PORT),) \
			$(if $(USER),-u $(USER),) \
			$(if $(PASSWORD),-P $(PASSWORD),) \
			$(if $(DB),-d $(DB),); \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f db-taxi
	rm -rf dist/
	@echo "Clean complete"
