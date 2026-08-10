# Developer & Engineering Implementation Guide

## 1. Prerequisites & Environment Setup

Ensure the following runtimes and tools are installed on your workstation:
- **Go:** `1.22+` (with `GOPATH` and Go modules enabled)
- **Node.js:** `v18+` or `v20+` LTS
- **npm:** `v9+` or `v10+`
- **Docker & Docker Compose:** (Optional, for PostgreSQL production orchestration)
- **Python:** `3.10+` with `python-docx` (for generating Word documentation)

---

## 2. Setting Up & Running the Application

### 2.1 Backend Execution
The backend automatically creates and seeds `log.db` on first execution:

```bash
cd backend

# Download Go module dependencies
go mod download

# Build and execute the binary
go build -o server main.go
./server
```

The Gin engine will start listening on `http://localhost:8080`.

---

### 2.2 Frontend Execution

```bash
cd frontend

# Install Node dependencies
npm install

# Create environment configuration file
echo "NEXT_PUBLIC_API_URL=http://localhost:8080/api" > .env.local

# Run development server
npm run dev
```

Visit `http://localhost:3000` in your web browser.

---

## 3. Implementation Workflow: Adding New Features

When extending the LOG platform, adhere strictly to the following implementation pattern:

### Step 1: Define Database Models (`backend/models/models.go`)
1. Create or modify GORM structs.
2. Add JSON serialization tags and GORM constraints (e.g. `gorm:"primaryKey"`, `gorm:"index"`).
3. Update `InitDB()` in `backend/database/db.go` to include the model in `AutoMigrate`.

### Step 2: Implement Backend API Handler (`backend/api/`)
1. Create handler functions in `handlers.go`, `admin.go`, or a new controller file.
2. Add input validation using Gin binding struct tags.
3. Protect the route with `AuthMiddleware(models.Role)` if restricted.
4. Mount the route on the Gin router in `backend/main.go`.

### Step 3: Define TypeScript Types (`frontend/src/lib/types.ts`)
1. Create matching TypeScript interfaces for API payloads.
2. Ensure strict nullability and optional parameter typing.

### Step 4: Integrate Frontend API Layer (`frontend/src/lib/api.ts`)
1. **Never use direct `fetch()`.** Always execute queries through `fetchWithCache(endpoint, options)`.
2. Ensure GET routes return cached data when offline.
3. Verify that mutating POST/PUT/DELETE calls are captured by `queueRequest()` when disconnected.

### Step 5: Construct UI Components & Pages (`frontend/src/app/`)
1. Build reusable components following the brand design system.
2. Use `framer-motion` for micro-animations.
3. Enforce **positive, supportive language** in all user-facing strings.

---

## 4. Running Tests & Quality Verification

### 4.1 Frontend Offline & Unit Tests
```bash
cd frontend
npm test
```

Expected output:
```
PASS  __tests__/api.test.ts
PASS  src/lib/syncExport.test.ts

Test Suites: 2 passed, 2 total
Tests:       11 passed, 11 total
```

Tests cover network-first fetching, cache fallback, offline mutation queueing with optimistic `202`, deduplication, cache TTL + legacy entries, and `.logsync` export/import.

### 4.2 Backend Compilation Verification
```bash
cd backend
go test ./...
go build -o server main.go
```

---

## 5. Docker Deployment

To launch the full-stack environment (backend + frontend, SQLite persisted in a named volume):

```bash
# Start services
docker-compose up -d

# Inspect running containers
docker-compose ps

# View service logs
docker-compose logs -f
```

> The Compose stack runs both tiers with the SQLite database persisted in the `log_data` volume — ideal for single-machine edge deployments. Set `JWT_SECRET` and `CORS_ORIGIN` in the backend service environment (already configured in `docker-compose.yml`).
