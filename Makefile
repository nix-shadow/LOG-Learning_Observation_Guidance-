.PHONY: dev build-backend build-frontend up down test clean

# Development
dev-backend:
	cd backend && go run main.go

dev-frontend:
	cd frontend && npm run dev

# Testing
test-frontend:
	cd frontend && npm run test

# Docker Orchestration
up:
	docker-compose up --build -d

down:
	docker-compose down

logs:
	docker-compose logs -f

# Clean up
clean:
	docker-compose down -v
	rm -rf backend/server
	rm -rf frontend/.next
