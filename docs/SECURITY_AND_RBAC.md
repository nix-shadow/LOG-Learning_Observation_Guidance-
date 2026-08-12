# Security & RBAC Guide

> Comprehensive security architecture for the LOG platform, designed for deployment in regions with low-connectivity and untrusted networks.

---

## 1. Multi-Tier Role-Based Access Control (RBAC)

LOG enforces a strict 3-tier RBAC hierarchy:

| Role | Level | Access | Route Prefix |
|------|-------|--------|-------------|
| `ADMIN` | Highest | Full system access, user management, role changes | `/api/admin/*` |
| `MODERATOR` | Middle | Class management, student progress, roster view | `/api/moderator/*` |
| `STUDENT` | Base | Dashboard, learning journey, courses, activities | `/api/*` (protected) |

### Privilege Escalation Rules

- **ADMIN** bypasses all role checks (universal access).
- **MODERATOR** can access both moderator-level and student-level endpoints.
- **STUDENT** can only access student-level endpoints.
- Role checks are enforced in `AuthMiddleware(requiredRole)` at the router group level — not per-handler.

### Implementation

```go
// backend/api/auth.go — AuthMiddleware
func AuthMiddleware(requiredRole models.Role) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... JWT validation ...
        if role == models.RoleAdmin {
            // Admin passes all checks
        } else if role == models.RoleModerator && (requiredRole == models.RoleStudent || requiredRole == models.RoleModerator) {
            // Moderator passes student and moderator checks
        } else if role != requiredRole {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
    }
}
```

---

## 2. Authentication & Token Security

### JWT Configuration

| Parameter | Value | Notes |
|-----------|-------|-------|
| **Algorithm** | HMAC-SHA256 (`HS256`) | Algorithm confusion attacks prevented by strict type check |
| **Secret** | `JWT_SECRET` env var | **Must be ≥ 32 characters.** Fatal startup if missing |
| **Expiry** | 72 hours | Includes `iat` (issued-at) for audit |
| **Claims** | `sub` (user ID), `role`, `jti` (unique token ID), `iat`, `exp` | `jti` powers server-side revocation |
| **Revocation** | `POST /api/auth/logout` | Adds `jti` to `TokenBlocklist`; expired entries purged on startup |

### Algorithm Confusion Prevention

```go
// Strictly enforce HMAC — reject RSA/ECDSA/None
if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
}
```

### Frontend Token Lifecycle

1. **On Login:** Token stored in `localStorage` and synced to a `SameSite=Lax` cookie for Next.js edge middleware.
2. **On Mount:** `AuthContext` decodes JWT, checks `exp` claim. Auto-logout if expired.
3. **Periodic Check:** Every 60 seconds, the active token is checked for expiry.
4. **On Logout:** `POST /api/auth/logout` revokes the token server-side, then localStorage, the token cookie, and the IndexedDB cache are cleared before redirecting to `/login`.

### OTP Security

- **Cryptographic generation:** 6-digit OTPs are produced with `crypto/rand` — no hardcoded demo codes.
- **Hashed at rest:** OTPs are bcrypt-hashed before persisting; plaintext OTPs are never stored.
- **Constant-time comparison** via `crypto/subtle.ConstantTimeCompare()` to prevent timing attacks.
- **Expiry:** OTPs expire after 5 minutes.
- **Cleanup:** Expired OTPs are purged on new OTP requests (both per-phone and globally for records >10 minutes old).

---

## 3. Rate Limiting

Auth endpoints (`/api/auth/*`) are protected by an in-memory per-IP token bucket rate limiter:

| Parameter | Value |
|-----------|-------|
| Max requests per window | 5 |
| Window duration | 1 minute |

A background goroutine (every 5 minutes) prunes stale IP entries to prevent memory leaks.

When exceeded, returns `429 Too Many Requests`:
```json
{
  "error": "Too many requests. Please wait before trying again."
}
```

---

## 4. DoS & Runtime Hardening

| Control | Value | Location |
|---------|-------|----------|
| Request body limit | 4 MB (`http.MaxBytesReader`) | `main.go` global middleware |
| Server read timeout | 15s | `http.Server` |
| Server write timeout | 30s | `http.Server` |
| Idle timeout | 120s | `http.Server` |
| Panic recovery | `gin.CustomRecovery` → clean `500` (no stack traces) | `main.go` |
| Graceful shutdown | `SIGINT`/`SIGTERM` with 5s context timeout | `main.go` |

---

## 5. Security Headers

All responses include the following headers (enforced globally in `main.go`):

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing |
| `X-Frame-Options` | `DENY` | Prevents clickjacking |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS filter |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(), payment=()` | Disables unused browser APIs |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' ...` | Restricts resource loading |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Forces HTTPS (1 year) |

---

## 6. CORS Policy

CORS is restricted to a single configured origin:

```bash
# Set in environment — defaults to http://localhost:6100
export CORS_ORIGIN=https://log.edu.np
```

- **Allowed Methods:** GET, POST, PUT, DELETE, OPTIONS
- **Allowed Headers:** Content-Type, Authorization, X-Request-ID
- **Exposed Headers:** X-Request-ID
- **Preflight Cache:** 86400 seconds (24 hours)

---

## 7. Frontend Route Protection

### Next.js Edge Middleware (`src/middleware.ts`)

Runs **before** any page render to prevent content flash:

| Route Pattern | Behavior |
|---------------|----------|
| `/dashboard`, `/learning/*`, `/courses`, `/observation`, `/guidance`, `/moderator`, `/admin` | Redirect to `/login?redirect=...` if no token cookie |
| `/login`, `/forgot-password` | Redirect to `/dashboard` if token cookie exists |

### Client-Side Guards

Pages also check roles in their `useEffect`:
- **Moderator pages:** Check `isModerator` from `AuthContext`.
- **Admin pages:** Check `isAdmin` from `AuthContext`.
- Unauthorized access shows a toast and redirects to `/dashboard`.

---

## 8. Audit Logging

Every request is logged with a structured audit trail:

```
[AUDIT] 2026-08-09T18:00:00Z | POST /api/activities/act-2/complete | status=200 | duration=15ms | user=user-123 | ip=192.168.1.1 | req=req-a1b2c3d4e5f67890
```

Fields:
- **Timestamp** (RFC3339)
- **Method + Path**
- **HTTP Status Code**
- **Response Duration**
- **Authenticated User ID** (empty for public routes)
- **Client IP**
- **Request ID** (unique per request, also returned in `X-Request-ID` header)

---

## 9. Database Security

### Password Hashing
- Algorithm: **bcrypt** with cost factor **14**
- Implemented in `HashPassword()` and `CheckPasswordHash()`

### Input Validation
All request bodies use Gin's struct binding with validation tags:
```go
Phone string `json:"phone" binding:"required,min=10,max=15"`
OTP   string `json:"otp" binding:"required,len=6"`
Email string `json:"email" binding:"required,email"`
```

### Soft Deletes
User and Activity models use GORM `DeletedAt` for soft deletion — data is never permanently removed.

### Transactions
`CompleteActivity` wraps all 4 database writes in a single `gorm.DB.Transaction()` to prevent inconsistent state.

---

## 10. Cryptographic ID Generation

All dynamically created entities (observations, guidance, request IDs) use `crypto/rand` for ID generation:

```go
func GenerateSecureID(prefix string) string {
    b := make([]byte, 8)
    rand.Read(b)
    return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}
```

This prevents sequential ID enumeration attacks.

---

## 11. Email Enumeration Prevention

The `/api/auth/forgot-password` endpoint always returns success regardless of whether the email exists:

```json
{
  "message": "If an account exists with this email, a password reset link has been sent."
}
```

---

## 12. Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes (prod) | Dev fallback with warning | HMAC signing key, ≥ 32 characters |
| `PORT` | No | `6101` | Backend listen port |
| `CORS_ORIGIN` | No | `http://localhost:6100` | Allowed CORS origin |
| `NEXT_PUBLIC_API_URL` | No | `http://localhost:6101/api` | Frontend API base URL |
