# REST API Specification & Endpoint Reference

## 1. Base Configuration
- **Base URL:** `http://localhost:6101/api/v1`
- **Default Port:** `6101`
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
- **Endpoint:** `/api/v1/auth/request-otp`
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
- **Endpoint:** `/api/v1/auth/verify-otp`
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
- **Endpoint:** `/api/v1/auth/forgot-password`
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
- **Endpoint:** `/api/v1/auth/logout`
- **Access:** Authenticated (any role)
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "message": "Logged out successfully."
  }
  ```

---

### 3.6 Login (Email & Password)
- **Method:** `POST`
- **Endpoint:** `/api/v1/auth/login`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "email": "aisha@example.com",
    "password": "supersecret"
  }
  ```
- **Validation Rules:** `email` required + valid format, `password` required.
- **Response (`200 OK`):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "user-123",
      "name": "Aisha Student",
      "email": "aisha@example.com",
      "role": "STUDENT"
    }
  }
  ```
- **Error Response (`401 Unauthorized`):** `{"detail": "Invalid email or password"}`

---

### 3.7 Register
Creates a new `STUDENT` account and auto-logs-in (issues a JWT).

- **Method:** `POST`
- **Endpoint:** `/api/v1/auth/register`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "name": "New Learner",
    "email": "learner@example.com",
    "password": "supersecret"
  }
  ```
- **Validation Rules:** `name` required, `email` required + valid, `password` required (min 8 chars).
- **Response (`201 Created`):**
  ```json
  {
    "message": "User registered successfully",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "usr-<random>",
      "name": "New Learner",
      "email": "learner@example.com",
      "role": "STUDENT"
    }
  }
  ```
- **Error Response (`409 Conflict`):** `{"detail": "An account with this email already exists"}`

---

### 3.8 Update Password
- **Method:** `PUT`
- **Endpoint:** `/api/v1/auth/password`
- **Access:** Authenticated (any role)
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
  ```json
  {
    "old_password": "current-password",
    "new_password": "new-password"
  }
  ```
- **Validation Rules:** `old_password` required, `new_password` required (min 8 chars).
- **Response (`200 OK`):**
  ```json
  {
    "message": "Password updated successfully"
  }
  ```

---

### 3.4 Google OAuth
Handles federated sign-in with Google OAuth. The Google `id_token` is verified server-side (issuer, audience, and signature via Google's public JWKS) against the configured `GOOGLE_CLIENT_ID` before any account is created or accessed.

- **Method:** `POST`
- **Endpoint:** `/api/v1/auth/google`
- **Access:** Public
- **Request Body:**
  ```json
  {
    "token": "<google-id-token>"
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
- **Error Response (`401 Unauthorized`):** invalid token, or `500` if `GOOGLE_CLIENT_ID` is not configured on the server.

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
- **Endpoint:** `/api/v1/dashboard`
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
- **Endpoint:** `/api/v1/learning-journey`
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
- **Endpoint:** `/api/v1/chart-data`
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
Records completion of a curriculum activity, increments the learner's streak, recalculates mastery score, and generates a positive observation and next-step guidance. When the client has just finished quiz knowledge checks inside the activity's micro-modules, it sends the **attempt facts** (`elapsed_seconds`, `correct_count`, `total_count`) so the score and guidance derive from real learning data instead of placeholders.

- **Method:** `POST`
- **Endpoint:** `/api/v1/activities/:id/complete`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body (optional — legacy clients may send an empty body):**
  ```json
  {
    "elapsed_seconds": 240,
    "correct_count": 3,
    "total_count": 4
  }
  ```
  - `correct_count` is clamped to `total_count`; accuracy = `correct_count / total_count`.
  - **Best-score semantics:** re-attempts never downgrade the recorded score. A first completion or an *improving* re-attempt bumps `attempts` and writes fresh observation/guidance; an equal or lower replay only refreshes `elapsed_seconds` (this keeps offline queue replays idempotent — no double progress, streak, or guidance).
  - **Guidance derivation** (supportive tone, derived from real accuracy):
    - accuracy ≥ 0.8 → strengths praise + next-step suggestion
    - accuracy 0.5–0.79 → practice-band suggestion ("this area could use more practice")
    - accuracy < 0.5 → foundations-band encouragement
    - no quiz data → legacy momentum encouragement (never a fabricated score)
- **Response (`200 OK`):**
  ```json
  {
    "message": "Activity marked as completed",
    "activity_id": "act-2",
    "attempt": {
      "accuracy": 0.75,
      "score": 75.0,
      "elapsed_seconds": 240
    },
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
- **Endpoint:** `/api/v1/courses`
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
- **Endpoint:** `/api/v1/activities/:id/modules`
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
Processes a batch of offline actions (`POST /activities/:id/complete`) uploaded from a `.logsync` file. Wrapped in a database transaction, scoped to the authenticated user's progress, and mirrors the online completion flow (creates observation + guidance). The optional `body` field carries the same attempt facts as the online endpoint, so offline completions land identical accuracy/score/attempt fields; malformed or unknown items are counted as failed and do not abort the batch.

- **Method:** `POST`
- **Endpoint:** `/api/v1/sync/bulk`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
  ```json
  {
    "version": "1.0",
    "timestamp": "2026-08-10T10:00:00.000Z",
    "data": [
      {
        "endpoint": "/activities/act-2/complete",
        "method": "POST",
        "body": "{\"elapsed_seconds\":200,\"correct_count\":4,\"total_count\":4}"
      }
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

### 5.1 Get Class Roster & Student Progress
Returns live student rosters, completion percentages, and attention flags for teachers.

- **Method:** `GET`
- **Endpoint:** `/api/v1/moderator/roster`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):**
  ```json
  {
    "class_name": "Logic 101: Discrete Structures",
    "active_students": 1,
    "needs_attention": 0,
    "assignments_due": 0,
    "roster": [
      {
        "id": "usr-0a1b2c3d4e5f6789",
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
- **Endpoint:** `/api/v1/admin/dashboard`
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
- **Endpoint:** `/api/v1/admin/users`
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
- **Endpoint:** `/api/v1/admin/users/:id/role`
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
- **Endpoint:** `/api/v1/admin/activities`
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
- **Audit:** every successful creation appends `activity.create` to the audit log.

---

### 6.5 Create Class
- **Method:** `POST`
- **Endpoint:** `/api/v1/admin/classes`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
  ```json
  {
    "name": "Grade 10 A",
    "grade": "10",
    "section": "A",
    "teacher_id": "mod-1"
  }
  ```
- **Validation Rules:** `name` required (2–120 chars), `grade` and `section` required, `teacher_id` required.
- **Response (`201 Created`):** the created class object (`id`, `name`, `grade`, `section`, `teacher_id`, `created_at`).
- **Audit:** appends `class.create`.

### 6.6 List Classes (Admin)
- **Method:** `GET`
- **Endpoint:** `/api/v1/admin/classes`
- **Response (`200 OK`):**
  ```json
  {
    "classes": [
      { "id": "cls-1", "name": "Grade 10 A", "grade": "10", "section": "A", "teacher_id": "mod-1", "created_at": "...", "member_count": 25 }
    ]
  }
  ```
- **Honesty rule:** `member_count` comes from real enrollment rows — never fabricated.

### 6.7 Enroll Students
- **Method:** `POST`
- **Endpoint:** `/api/v1/admin/classes/:id/enroll`
- **Request Body:** `{ "user_ids": ["user-123", "user-456"] }`
- **Behavior:** only users with role `STUDENT` are enrolled; staff/moderator IDs are silently skipped. Duplicate enrollments are idempotent (ON CONFLICT DO NOTHING).
- **Response (`200 OK`):** `{ "message": "Students enrolled", "member_count": 25 }`
- **Audit:** appends `class.enroll` with the new member count.

### 6.8 Class Roster & Unenroll
- **Method:** `GET /api/v1/admin/classes/:id/roster` — enrolled students of a class.
- **Method:** `DELETE /api/v1/admin/classes/:id/enroll/:user_id` — removes a student (audit: `class.unenroll`).

### 6.9 Audit Log
- **Method:** `GET /api/v1/admin/audit-log?limit=50`
- **Response (`200 OK`):** `{ "audit_logs": [{ "id": 1, "user_id": "admin-1", "action": "user.role_change", "detail": "user-123 -> MODERATOR", "ip": "127.0.0.1", "created_at": "..." }] }`
- **Actions recorded:** `user.role_change`, `activity.create`, `class.create`, `class.enroll`, `class.unenroll`, `announcement.create`, `assignment.create`, `export.students_csv`, `auth.logout_all`.

### 6.10 Export Students (CSV)
- **Method:** `GET /api/v1/admin/export/students.csv`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Response (`200 OK`):** `text/csv` attachment with header `name,email,phone,class`. Only real enrolled students appear; learners in multiple classes get a `; `-separated class list. Empty enrollments produce a header-only file (honest empty state).

---

## 7. School Operations (Classes, Announcements, Assignments)

### 7.1 List My Classes (Moderator)
- **Method:** `GET /api/v1/moderator/classes`
- **Response:** classes where the caller is the assigned `teacher_id`, each with a real `member_count`.

### 7.2 Create Assignment
- **Method:** `POST /api/v1/moderator/classes/:id/assignments`
- **Request Body:**
  ```json
  {
    "title": "Homework 1 — Truth Tables",
    "description": "Solve exercises 1–5",
    "activity_id": "act-2",
    "due_date": "2026-08-25T23:59:00Z"
  }
  ```
- **Authorization:** only the class's own teacher (or an admin) may create assignments — others receive `403`.
- **Validation:** `title` required (3–200 chars); `due_date` must be RFC 3339 if present.
- **Response (`201 Created`):** the created assignment.
- **Audit:** appends `assignment.create`.

### 7.3 List Assignments & Submissions (Moderator)
- **Method:** `GET /api/v1/moderator/classes/:id/assignments` — assignments for a class, ordered by due date.
- **Method:** `GET /api/v1/moderator/classes/:id/assignments/:assignment_id/submissions` — per-learner submissions with `learner_id`, `note`, `submitted_at`.

### 7.4 My Assignments (Student)
- **Method:** `GET /api/v1/assignments`
- **Response:** assignments for every class the learner is enrolled in, each with the live submission count. Offline-cacheable GET.

### 7.5 Submit Assignment (Student)
- **Method:** `POST /api/v1/assignments/:assignment_id/submit`
- **Request Body:** `{ "note": "My answers are ready" }` (required, max 4000 chars)
- **Authorization:** only learners enrolled in the assignment's class may submit — others receive `403`.
- **Idempotency:** `(assignment_id, learner_id)` is a unique pair; resubmission (including offline queue replays) updates the same row instead of duplicating.
- **Offline:** uses the standard mutating path — queued when offline, flushed on reconnect.

### 7.6 Announcements
- **Method:** `GET /api/v1/announcements?limit=10` — any authenticated role; newest first.
- **Method:** `POST /api/v1/admin/announcements` or `POST /api/v1/moderator/announcements` — `{ "title": "...", "body": "..." }`; `title` required (3–200 chars).
- **Audit:** appends `announcement.create`.

### 7.7 Log Out on All Devices
- **Method:** `POST /api/v1/auth/logout-all`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Behavior:** records a `UserRevocation` row with the current timestamp. Any token whose `iat` predates it is rejected by `AuthMiddleware` (checked via `RevokedBefore`), even before natural expiry. Subsequent logins mint tokens with a newer `iat` and work normally.
- **Response (`200 OK`):** `{ "message": "Logged out on all devices" }`
- **Audit:** appends `auth.logout_all`.
