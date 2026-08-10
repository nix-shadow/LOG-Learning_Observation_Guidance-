# REST API Specification & Endpoint Reference

## 1. Base Configuration
- **Base URL:** `http://localhost:8080/api`
- **Default Port:** `8080`
- **Content Type:** `application/json`
- **Authentication:** `Bearer <JWT_TOKEN>` in `Authorization` header

---

## 2. Global Security Headers
Every API response from the Go Gin backend includes the following HTTP headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'self'; ...`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Access-Control-Allow-Origin: <CORS_ORIGIN>` (origin-restricted — NEVER `*` by default)
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID`
- `Access-Control-Expose-Headers: X-Request-ID`

---

## 3. Authentication Endpoints

### 3.1 Request OTP
Initiates phone number verification by generating a 6-digit OTP.

- **Method:** `POST`
- **Endpoint:** `/api/auth/request-otp`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "phone": "+9779800000000"
  }
  ```
- **Validation Rules:** `phone` required, minimum 10 characters, maximum 15 characters.
- **Response (`200 OK`):**
  ```json
  {
    "message": "OTP sent"
  }
  ```
- **Error Response (`400 Bad Request`):**
  ```json
  {
    "error": "Invalid phone number format"
  }
  ```

---

### 3.2 Verify OTP
Verifies the submitted 6-digit OTP, creates the user account if new, and issues an HMAC-SHA256 JWT token.

- **Method:** `POST`
- **Endpoint:** `/api/auth/verify-otp`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "phone": "+9779800000000",
    "otp": "123456"
  }
  ```
- **Validation Rules:** `phone` required, `otp` required (exact length 6).
- **Response (`200 OK`):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "user-123",
      "name": "Aisha Student",
      "email": "aisha@example.com",
      "phone": "+9779800000000",
      "role": "STUDENT",
      "is_verified": true,
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  }
  ```
- **Error Responses:**
  - `400 Bad Request`: `{"error": "Invalid request"}`
  - `401 Unauthorized`: `{"error": "Invalid or expired OTP"}`

---

### 3.3 Forgot Password
Dispatches password reset verification link to registered email.

- **Method:** `POST`
- **Endpoint:** `/api/auth/forgot-password`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "email": "learner@example.com"
  }
  ```
- **Validation Rules:** `email` required, valid email format.
- **Response (`200 OK`):**
  ```json
  {
    "message": "If an account exists with this email, a password reset link has been sent."
  }
  ```
- **Security Note:** Always returns success regardless of email existence to prevent email enumeration.

---

### 3.5 Logout (JWT Revocation)
Invalidates the caller's JWT by adding its `jti` to the `TokenBlocklist`. Subsequent requests with that token receive `401`.

- **Method:** `POST`
- **Endpoint:** `/api/auth/logout`
- **Access:** Authenticated (any role)
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "message": "Logged out successfully."
  }
  ```

---

### 3.4 Google OAuth Simulation
Handles federated sign-in with Google OAuth.

- **Method:** `POST`
- **Endpoint:** `/api/auth/google`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "email": "learner@gmail.com",
    "name": "Google Learner"
  }
  ```
- **Response (`200 OK`):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "user-g-1723225200",
      "name": "Google Learner",
      "email": "learner@gmail.com",
      "role": "STUDENT"
    }
  }
  ```

---

## 4. Student Endpoints (Protected: Student / Moderator / Admin)

### 4.1 Health Check / Ping
- **Method:** `GET`
- **Endpoint:** `/api/ping`
- **Access:** Public / Student
- **Response (`200 OK`):**
  ```json
  {
    "message": "pong"
  }
  ```

---

### 4.2 Get Dashboard Data
Fetches the learner profile, streak metrics, active guidance, and recent observations.

- **Method:** `GET`
- **Endpoint:** `/api/dashboard`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "learner": {
      "id": "user-123",
      "name": "Aisha Student",
      "email": "aisha@example.com",
      "phone": "+9779800000000",
      "role": "STUDENT",
      "is_verified": true,
      "created_at": "2026-08-09T17:00:00Z"
    },
    "progress": {
      "learner_id": "user-123",
      "total_topics": 10,
      "completed": 2,
      "current_streak": 3,
      "overall_score": 85.5
    },
    "activities": [
      {
        "id": "act-1",
        "title": "Introduction to Logic",
        "description": "Basic concepts.",
        "status": "Completed",
        "topic": "Logic",
        "order": 1
      },
      {
        "id": "act-2",
        "title": "Boolean Algebra",
        "description": "AND, OR, NOT.",
        "status": "In progress",
        "topic": "Logic",
        "order": 2
      }
    ],
    "observations": [
      {
        "id": "obs-1",
        "learner_id": "user-123",
        "category": "strengths",
        "text": "Strong grasp of Boolean Algebra.",
        "created_at": "2026-08-09T17:00:00Z"
      },
      {
        "id": "obs-2",
        "learner_id": "user-123",
        "category": "consistency",
        "text": "Studying consistently for 3 days.",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ],
    "guidance": [
      {
        "id": "gui-1",
        "learner_id": "user-123",
        "text": "Continue Boolean Algebra.",
        "action": "/learning/act-2",
        "type": "next_step",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ]
  }
  ```

---

### 4.3 Get Learning Journey
Returns all curriculum modules ordered sequentially.

- **Method:** `GET`
- **Endpoint:** `/api/learning-journey`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "activities": [
      {
        "id": "act-1",
        "title": "Introduction to Logic",
        "description": "Basic concepts.",
        "status": "Completed",
        "topic": "Logic",
        "order": 1
      },
      {
        "id": "act-2",
        "title": "Boolean Algebra",
        "description": "AND, OR, NOT.",
        "status": "In progress",
        "topic": "Logic",
        "order": 2
      }
    ]
  }
  ```

---

### 4.4 Get Chart Telemetry Data
Returns 7-day engagement metrics and score trends for Recharts rendering.

- **Method:** `GET`
- **Endpoint:** `/api/chart-data`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "activity_data": [
      { "name": "Mon", "score": 65, "duration": 20 },
      { "name": "Tue", "score": 70, "duration": 25 },
      { "name": "Wed", "score": 68, "duration": 15 },
      { "name": "Thu", "score": 75, "duration": 30 },
      { "name": "Fri", "score": 85, "duration": 45 },
      { "name": "Sat", "score": 82, "duration": 40 },
      { "name": "Sun", "score": 88, "duration": 50 }
    ]
  }
  ```

---

### 4.5 Complete Activity Module
Records completion of a curriculum activity, increments the learner's streak, recalculates mastery score, and generates a positive observation and next-step guidance.

- **Method:** `POST`
- **Endpoint:** `/api/activities/:id/complete`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "message": "Activity marked as completed",
    "activity_id": "act-2",
    "progress": {
      "learner_id": "user-123",
      "completed": 3,
      "current_streak": 4,
      "overall_score": 88.0
    },
    "observation": {
      "id": "obs-1723225200",
      "category": "strengths",
      "text": "Demonstrated excellent focus and successfully completed Boolean Algebra."
    },
    "guidance": {
      "id": "gui-1723225200",
      "type": "next_step",
      "text": "Great momentum! Continue to the next practice module to reinforce your logic skills.",
      "action": "/learning"
    }
  }
  ```

---

### 4.6 Get Courses Catalog
Returns the comprehensive course catalog (from the `courses` table) for offline and online browsing.

- **Method:** `GET`
- **Endpoint:** `/api/courses`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Query Params:** `page` (default 1), `limit` (default 10, max 100)
- **Response (`200 OK`):**
  ```json
  {
    "courses": [
      {
        "id": "course-1",
        "title": "Fundamentals of Logic & Gates",
        "category": "Computer Science",
        "difficulty": "Beginner",
        "duration": "3 hours",
        "rating": 4.9,
        "enrolled": 1250
      }
    ],
    "pagination": { "page": 1, "limit": 10, "total": 5 }
  }
  ```

---

### 4.7 Get Activity Micro-Modules
Returns the bite-sized `MicroModule` entries for an activity, ordered sequentially for the viewer.

- **Method:** `GET`
- **Endpoint:** `/api/activities/:id/modules`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "activity": { "id": "act-2", "title": "Boolean Algebra", "status": "In progress" },
    "modules": [
      { "id": "mm-3", "activity_id": "act-2", "title": "The AND Operator", "content_text": "...", "order": 1 }
    ],
    "total": 1
  }
  ```
- **Error Response (`404 Not Found`):** `{"error": "Activity not found"}`

---

### 4.8 Bulk Offline Sync (Sneakernet)
Processes a batch of offline actions (`POST /activities/:id/complete`) uploaded from a `.logsync` file. Wrapped in a database transaction, scoped to the authenticated user's progress, and mirrors the online completion flow (creates observation + guidance).

- **Method:** `POST`
- **Endpoint:** `/api/sync/bulk`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
  ```json
  {
    "version": "1.0",
    "timestamp": "2026-08-10T10:00:00.000Z",
    "data": [
      { "endpoint": "/activities/act-2/complete", "method": "POST", "body": "{}" }
    ]
  }
  ```
- **Response (`200 OK`):**
  ```json
  {
    "message": "Successfully synced 1 offline actions.",
    "count": 1
  }
  ```
- **Error Response (`400 Bad Request`):** `{"error": "Invalid sync payload format"}`

---

## 5. Moderator Endpoints (Protected: Moderator / Admin)

### 5.1 Get Moderator Classes
- **Method:** `GET`
- **Endpoint:** `/api/moderator/classes`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "message": "Moderator classes data"
  }
  ```

---

### 5.2 Get Class Roster & Student Progress
Returns live student rosters, completion percentages, and attention flags for teachers.

- **Method:** `GET`
- **Endpoint:** `/api/moderator/roster`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "class_name": "Logic 101: Discrete Structures",
    "active_students": 124,
    "needs_attention": 8,
    "assignments_due": 3,
    "roster": [
      {
        "id": "user-123",
        "name": "Aisha Student",
        "completion": 85,
        "streak": 4,
        "status": "Active",
        "last_active": "Jan 02"
      }
    ],
    "pagination": { "page": 1, "limit": 20, "total": 124 }
  }
  ```
  All `needs_attention` and `assignments_due` values are computed from live database rows (students with a zero streak, activities in progress).

- **Error Response (`403 Forbidden`):**
  ```json
  {
    "error": "Insufficient permissions"
  }
  ```

---

## 6. Admin Endpoints (Protected: Admin Only)

### 6.1 Get Admin Dashboard
- **Method:** `GET`
- **Endpoint:** `/api/admin/dashboard`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "analytics": {
      "total_users": 15,
      "active_daily": 7,
      "total_completions": 40
    },
    "recent_users": [
      {
        "id": "admin-1",
        "name": "Principal Skinner",
        "email": "admin@log.edu",
        "phone": "1000000000",
        "role": "ADMIN",
        "is_verified": true,
        "created_at": "2026-08-09T17:00:00Z"
      }
    ]
  }
  ```

---

### 6.2 Get All Users
- **Method:** `GET`
- **Endpoint:** `/api/admin/users`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "users": [
      {
        "id": "admin-1",
        "name": "Principal Skinner",
        "email": "admin@log.edu",
        "phone": "1000000000",
        "role": "ADMIN",
        "is_verified": true
      }
    ]
  }
  ```

---

### 6.3 Update User Role
- **Method:** `PUT`
- **Endpoint:** `/api/admin/users/:id/role`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
  ```json
  {
    "role": "MODERATOR"
  }
  ```
- **Response (`200 OK`):**
  ```json
  {
    "message": "Role updated",
    "user": {
      "id": "user-123",
      "name": "Aisha Student",
      "role": "MODERATOR"
    }
  }
  ```

---

### 6.4 Create Learning Activity
- **Method:** `POST`
- **Endpoint:** `/api/admin/activities`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body** (client may NOT inject `id`, `order`, `status`, or `created_at` — all server-managed):
  ```json
  {
    "title": "De Morgan's Laws",
    "description": "Negation of conjunctions and disjunctions.",
    "topic": "Logic",
    "difficulty": "Intermediate",
    "prerequisites": "",
    "content_json": "{}"
  }
  ```
- **Validation Rules:** `title` required (3–200 chars), `description` required, `topic` required, `difficulty` must be one of `Beginner`, `Intermediate`, `Advanced`.
- **Response (`201 Created`):**
  ```json
  {
    "id": "act-<random>",
    "title": "De Morgan's Laws",
    "status": "Not Started",
    "order": 3,
    "created_at": "2026-08-09T17:00:00Z"
  }
  ```
