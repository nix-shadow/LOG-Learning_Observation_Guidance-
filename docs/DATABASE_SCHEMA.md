# Database Schema & Data Models

## 1. Overview
The database layer uses GORM with an auto-migrating schema. The primary database file for local development is SQLite (`backend/data/log.db`). Production environments can be pointed to PostgreSQL using standard GORM connection drivers.

---

## 2. Entity Relationship Diagram

```
+------------------------------------+
|               User                 |
+------------------------------------+
| PK: ID (string)                    |
|     Name (string)                  |
|     Email (string, uniqueIndex)    |
|     Phone (string, uniqueIndex)    |
|     PasswordHash (string, hidden)  |
|     Role (string: ADMIN/MOD/STUD)  |
|     IsVerified (bool)              |
|     CreatedAt / UpdatedAt          |
|     DeletedAt (soft delete index)  |
+------------------------------------+
         | 1
         |
         +-----------------+-----------------+
         | 1               | 1               | 1
         v                 v                 v
+-----------------+ +---------------+ +---------------+
|    Progress     | |  Observation  | |   Guidance    |
+-----------------+ +---------------+ +---------------+
| PK: LearnerID   | | PK: ID        | | PK: ID        |
| TotalTopics     | | LearnerID     | | LearnerID     |
| Completed       | | Category      | | Text          |
| CurrentStreak   | | Text          | | Action (URL)  |
| OverallScore    | | CreatedAt     | | Type          |
+-----------------+ +---------------+ | CreatedAt     |
                                      +---------------+

+------------------------------------+  +--------------------+
|              Activity              |  |     OTPRecord      |
+------------------------------------+  +--------------------+
| PK: ID (string)                    |  | PK: Phone (string) |
|     Title (string)                 |  |     OTP (string)   |
|     Description (string)           |  |     ExpiresAt      |
|     Status (Completed/In progress) |  +--------------------+
|     Topic (string)                 |
|     Order (int)                    |
|     ContentJSON (text)             |
|     CreatedAt / DeletedAt          |
+------------------------------------+

+-------------------+  +--------------------+  +------------------+
|    MicroModule    |  |      Course        |  |  DailyActivity   |
+-------------------+  +--------------------+  +------------------+
| PK: ID (string)   |  | PK: ID (string)    |  | PK: ID (string)  |
|     ActivityID (FK)| |     Title (string)  |  |     LearnerID    |
|     Title         |  |     Category        |  |     Date         |
|     ContentText   |  |     Difficulty      |  |     DayName      |
|     MediaURL      |  |     Duration        |  |     Score        |
|     Order         |  |     Rating          |  |     Duration     |
+-------------------+  |     Enrolled        |  +------------------+
                       +--------------------+

+--------------------+
|  TokenBlocklist    |
+--------------------+
| PK: JTI (string)   |
|     UserID         |
|     ExpiresAt      |
|     RevokedAt      |
+--------------------+
```

---

## 3. GORM Model Structs (`backend/internal/domain/domain.go`)

### 3.1 User Model
```go
type Role string

const (
    RoleStudent   Role = "STUDENT"
    RoleModerator Role = "MODERATOR" // Teacher
    RoleAdmin     Role = "ADMIN"     // Principal/HOD
)

type User struct {
    ID           string         `json:"id" gorm:"primaryKey"`
    Name         string         `json:"name"`
    Email        string         `json:"email" gorm:"uniqueIndex"`
    Phone        string         `json:"phone" gorm:"uniqueIndex"`
    PasswordHash string         `json:"-"`
    Role         Role           `json:"role"`
    IsVerified   bool           `json:"is_verified"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### 3.2 OTPRecord Model
```go
type OTPRecord struct {
    Phone     string    `json:"phone" gorm:"primaryKey"`
    OTP       string    `json:"otp"`
    ExpiresAt time.Time `json:"expires_at"`
    Attempts  int       `json:"attempts"` // failed verify attempts — 5 fails invalidates the OTP
}
```

### 3.3 Activity Model
```go
type Activity struct {
    ID            string         `json:"id" gorm:"primaryKey"`
    Title         string         `json:"title"`
    Description   string         `json:"description"`
    Topic         string         `json:"topic"`
    Order         int            `json:"order"`
    ContentJSON   string         `json:"content_json"`
    Difficulty    string         `json:"difficulty"`    // Beginner, Intermediate, Advanced
    Prerequisites string         `json:"prerequisites"` // comma-separated Activity IDs
    CreatedAt     time.Time      `json:"created_at"`
    DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
```

Per-learner status lives in `LearnerActivity` (see §3.3a) — the global `Status` column was removed from `Activity`.

### 3.3a LearnerActivity Model
Tracks each learner's per-activity state. The attempt columns (`Score`, `Accuracy`, `ElapsedSeconds`, `Attempts`) are filled from **real quiz facts** sent by the client on completion (`correct_count` / `total_count` / `elapsed_seconds`) — the frontend must never fabricate these. Best-score semantics: `Score`/`Accuracy` only improve; `Attempts` increments on first completion or an improving re-attempt (equal/lower replays, e.g. offline queue flushes, only refresh `ElapsedSeconds` so replays stay idempotent).
```go
type LearnerActivity struct {
    LearnerID      string    `json:"learner_id" gorm:"primaryKey"`
    ActivityID     string    `json:"activity_id" gorm:"primaryKey"`
    Status         string    `json:"status"` // e.g. "Completed", "Pending", "In Progress"
    CompletedAt    time.Time `json:"completed_at"`
    Score          float64   `json:"score"` // best attempt score (0-100), derived from real quiz accuracy
    Accuracy       float64   `json:"accuracy"`
    ElapsedSeconds int       `json:"elapsed_seconds"`
    Attempts       int       `json:"attempts"`
}
```

### 3.4 Progress Model
```go
type Progress struct {
    LearnerID        string    `json:"learner_id" gorm:"primaryKey"`
    TotalTopics      int       `json:"total_topics"`
    Completed        int       `json:"completed"`
    CurrentStreak    int       `json:"current_streak"`
    OverallScore     float64   `json:"overall_score"`
    LastActivityDate time.Time `json:"last_activity_date"` // streak math is date-aware
}
```

### 3.5 Observation Model
```go
type Observation struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    LearnerID string    `json:"learner_id" gorm:"index"`
    Category  string    `json:"category"` // strengths, areas needing improvement, consistency
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 3.6 Guidance Model
```go
type Guidance struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    LearnerID string    `json:"learner_id" gorm:"index"`
    Text      string    `json:"text"`
    Action    string    `json:"action"` // e.g. "/learning/act-2"
    Type      string    `json:"type"`   // next_step, practice, insight
    CreatedAt time.Time `json:"created_at"`
}
```

### 3.7 MicroModule Model
Micro-lessons rendered sequentially inside an activity. Quiz fields (`Question`, `Options`, `CorrectIndex`, `Explanation`) are optional — when present the viewer renders a knowledge check that must be answered correctly before advancing, and the learner's first-try correctness feeds the completion attempt facts. `Options` is stored as a JSON-serialized column via GORM's `serializer:json`.
```go
type MicroModule struct {
    ID           string    `json:"id" gorm:"primaryKey"`
    ActivityID   string    `json:"activity_id" gorm:"index"`
    Title        string    `json:"title"`
    ContentText  string    `json:"content_text"` // extremely compressed text
    MediaURL     string    `json:"media_url"`    // optional low-res WebP image
    Question     string    `json:"question"`     // optional knowledge check
    Options      []string  `json:"options" gorm:"serializer:json"`
    CorrectIndex int       `json:"correct_index"`
    Explanation  string    `json:"explanation"` // supportive feedback shown after answering
    Order        int       `json:"order"`
    CreatedAt    time.Time `json:"created_at"`
}
```

### 3.8 Course Model
```go
type Course struct {
    ID         string         `json:"id" gorm:"primaryKey"`
    Title      string         `json:"title"`
    Category   string         `json:"category"`
    Difficulty string         `json:"difficulty"`
    Duration   string         `json:"duration"`
    Rating     float64        `json:"rating"`
    Enrolled   int            `json:"enrolled"`
    CreatedAt  time.Time      `json:"created_at"`
    DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### 3.9 DailyActivity Model
```go
type DailyActivity struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    LearnerID string    `json:"learner_id" gorm:"index"`
    Date      time.Time `json:"date" gorm:"index"`
    DayName   string    `json:"name"` // e.g. "Mon"
    Score     float64   `json:"score"`
    Duration  int       `json:"duration"` // in minutes
}
```

### 3.10 TokenBlocklist Model (JWT Revocation)
```go
type TokenBlocklist struct {
    JTI       string    `json:"jti" gorm:"primaryKey"`   // JWT ID claim
    UserID    string    `json:"user_id" gorm:"index"`    // which user revoked
    ExpiresAt time.Time `json:"expires_at" gorm:"index"` // mirrors JWT exp — for cleanup
    RevokedAt time.Time `json:"revoked_at"`
}
```

### 3.11 Class Model (School Operations)
```go
type Class struct {
    ID        string         `json:"id" gorm:"primaryKey"`
    Name      string         `json:"name" gorm:"not null"` // e.g. "Grade 10 A"
    Grade     string         `json:"grade"`
    Section   string         `json:"section"`
    TeacherID string         `json:"teacher_id" gorm:"index"` // MODERATOR who owns the class
    CreatedAt time.Time      `json:"created_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### 3.12 ClassMember Model (Enrollment)
```go
type ClassMember struct {
    ClassID  string    `json:"class_id" gorm:"primaryKey"`
    UserID   string    `json:"user_id" gorm:"primaryKey"`
    JoinedAt time.Time `json:"joined_at"`
}
```
Only `STUDENT`-role users can be enrolled — the repo filters by role inside a transaction.

### 3.13 Announcement Model
```go
type Announcement struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Title     string    `json:"title" gorm:"not null"`
    Body      string    `json:"body"`
    AuthorID  string    `json:"author_id" gorm:"index"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.14 Assignment Model
```go
type Assignment struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    ClassID     string    `json:"class_id" gorm:"index"`
    Title       string    `json:"title" gorm:"not null"`
    Description string    `json:"description"`
    ActivityID  string    `json:"activity_id"` // optional linked activity
    DueDate     time.Time `json:"due_date"`
    CreatedBy   string    `json:"created_by" gorm:"index"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 3.15 Submission Model
```go
type Submission struct {
    ID           string    `json:"id" gorm:"primaryKey"`
    AssignmentID string    `json:"assignment_id" gorm:"uniqueIndex:idx_sub_assignment_learner"`
    LearnerID    string    `json:"learner_id" gorm:"uniqueIndex:idx_sub_assignment_learner"`
    Note         string    `json:"note"`
    SubmittedAt  time.Time `json:"submitted_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```
The unique pair `(assignment_id, learner_id)` makes resubmission and offline replays idempotent (upsert via `ON CONFLICT DO UPDATE`).

### 3.16 AuditLog Model (Append-Only)
```go
type AuditLog struct {
    ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID    string    `json:"user_id" gorm:"index"`
    Action    string    `json:"action" gorm:"index"`
    Detail    string    `json:"detail"`
    IP        string    `json:"ip"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 3.17 UserRevocation Model (Log Out on All Devices)
```go
type UserRevocation struct {
    UserID        string    `json:"user_id" gorm:"primaryKey"`
    RevokedBefore time.Time `json:"revoked_before"`
}
```
`AuthMiddleware` rejects any token whose `iat` is older than `RevokedBefore`, regardless of remaining expiry.

---

## 4. Auto-Migration & Seeding Pipeline (`backend/database/db.go`)

On server startup, `InitDB()` runs `AutoMigrate`:

```go
DB.AutoMigrate(
    &models.User{},
    &models.OTPRecord{},
    &models.Activity{},
    &models.Progress{},
    &models.Observation{},
    &models.Guidance{},
    &models.Course{},
    &models.DailyActivity{},
    &models.MicroModule{},
    &models.TokenBlocklist{},
    &models.Class{},
    &models.ClassMember{},
    &models.Announcement{},
    &models.Assignment{},
    &models.Submission{},
    &models.AuditLog{},
    &models.UserRevocation{},
)
```

Expired blocklist entries are purged on startup to keep the table lean.

### Default Seed Data:
- **Admin User:** `admin-1` (Principal Skinner, `admin@log.edu`, Role: `ADMIN`)
- **Moderator User:** `mod-1` (Teacher Edna, `teacher@log.edu`, Role: `MODERATOR`)
- **Student User:** `user-123` (Aisha Student, `aisha@example.com`, Role: `STUDENT`)
- **Progress:** 10 total topics, 2 completed, 3-day streak, 85.5% overall score.
- **Activities:**
  - `act-1`: "Introduction to Logic" (Completed)
  - `act-2`: "Boolean Algebra" (In progress)
- **Micro-Modules:** 2 for `act-1`, 3 for `act-2` (seeded independently of users, so existing databases receive them on next startup).
- **Courses:** 5 seeded catalog entries (CS / Frontend / Backend / Design categories).
- **DailyActivity:** 7 seeded chart records (Mon–Sun) for `user-123`.
- **Class:** `cls-1` "Grade 10 A" (teacher `mod-1`, student `user-123` enrolled) — seeded independently of users so existing databases receive a working demo class.
- **Announcement:** `ann-1` "Welcome to LOG" by `admin-1`.
- **Observations:** "Strong grasp of Boolean Algebra", "Studying consistently for 3 days".
- **Guidance:** "Continue Boolean Algebra" (Action: `/learning/act-2`).
