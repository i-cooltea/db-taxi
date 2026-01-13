.PHONY: help build run test migrate migrate-status migrate-version clean diagnose-jobs fix-jobs test-job-engine diagnose-engine

# Default target
help:
	@echo "DB-Taxi Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build              Build the application"
	@echo "  run                Run the application"
	@echo "  test               Run tests"
	@echo "  test-job-engine    Test job engine startup"
	@echo "  diagnose-engine    Diagnose job engine issues"
	@echo "  check-engine       Check job engine status (requires running server)"
	@echo "  migrate            Run database migrations"
	@echo "  migrate-status     Show migration status"
	@echo "  migrate-version    Show current migration version"
	@echo "  diagnose-jobs      Diagnose stuck sync jobs"
	@echo "  fix-jobs           Fix stuck sync jobs"
	@echo "  clean              Clean build artifacts"
	@echo ""
	@echo "Migration examples:"
	@echo "  make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb"
	@echo "  make migrate-status CONFIG=config.yaml"
	@echo ""
	@echo "Job troubleshooting examples:"
	@echo "  make diagnose-jobs CONFIG=configs/config.yaml"
	@echo "  make fix-jobs CONFIG=configs/config.yaml TIMEOUT=30"
	@echo "  make test-job-engine CONFIG=configs/config.yaml"
	@echo "  make diagnose-engine"

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

# Diagnose stuck sync jobs
diagnose-jobs:
	@echo "Diagnosing stuck sync jobs..."
	@if [ -n "$(CONFIG)" ]; then \
		go run cmd/fix-jobs/main.go -config $(CONFIG) -dry-run; \
	else \
		go run cmd/fix-jobs/main.go -config configs/config.yaml -dry-run; \
	fi

# Fix stuck sync jobs
fix-jobs:
	@echo "Fixing stuck sync jobs..."
	@if [ -n "$(CONFIG)" ]; then \
		go run cmd/fix-jobs/main.go -config $(CONFIG) $(if $(TIMEOUT),-timeout $(TIMEOUT),); \
	else \
		go run cmd/fix-jobs/main.go -config configs/config.yaml $(if $(TIMEOUT),-timeout $(TIMEOUT),); \
	fi

# Test job engine startup
test-job-engine:
	@echo "Testing job engine startup..."
	@if [ -n "$(CONFIG)" ]; then \
		go run cmd/test-job-engine/main.go -config $(CONFIG); \
	else \
		go run cmd/test-job-engine/main.go; \
	fi

# Diagnose job engine issues
diagnose-engine:
	@echo "Diagnosing job engine..."
	@chmod +x scripts/diagnose-job-engine.sh
	@./scripts/diagnose-job-engine.sh

# Check job engine status (requires running server)
check-engine:
	@echo "Checking job engine status..."
	@chmod +x scripts/check-job-engine-status.sh
	@./scripts/check-job-engine-status.sh
