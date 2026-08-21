.PHONY: dev-backend dev-frontend build-backend build-frontend build budget test test-backend test-frontend lint lint-backend lint-frontend up down logs clean

# Development
dev-backend:
	cd backend && go run main.go

dev-frontend:
	cd frontend && npm run dev

# Build
build-backend:
	cd backend && go build -o server main.go

build-frontend:
	cd frontend && npm run build

build: build-backend build-frontend

# RESPECT budget gate (WP-4.2): fail when any route's First Load JS > 500 kB
budget:
	cd frontend && node scripts/check-budget.mjs

# Testing
test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm run test

test: test-backend test-frontend

# Linting
lint-backend:
	cd backend && golangci-lint run ./...

lint-frontend:
	cd frontend && npm run lint

lint: lint-backend lint-frontend

# Docker Orchestration
up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

# Clean up
clean:
	docker compose down -v
	rm -rf backend/server
	rm -rf frontend/.next