# Database Schema & Data Models

## 1. Overview
The database layer uses GORM with an auto-migrating schema. The primary database file for local development is SQLite (`backend/log.db`). Production environments can be pointed to PostgreSQL using standard GORM connection drivers.

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

## 3. GORM Model Structs (`backend/models/models.go`)

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
}
```

### 3.3 Activity Model
```go
type Activity struct {
    ID          string         `json:"id" gorm:"primaryKey"`
    Title       string         `json:"title"`
    Description string         `json:"description"`
    Status      string         `json:"status"` // Completed, In progress, Pending
    Topic       string         `json:"topic"`
    Order       int            `json:"order"`
    ContentJSON string         `json:"content_json"`
    CreatedAt   time.Time      `json:"created_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### 3.4 Progress Model
```go
type Progress struct {
    LearnerID     string  `json:"learner_id" gorm:"primaryKey"`
    TotalTopics   int     `json:"total_topics"`
    Completed     int     `json:"completed"`
    CurrentStreak int     `json:"current_streak"`
    OverallScore  float64 `json:"overall_score"`
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
```go
type MicroModule struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    ActivityID  string    `json:"activity_id" gorm:"index"`
    Title       string    `json:"title"`
    ContentText string    `json:"content_text"` // extremely compressed text
    MediaURL    string    `json:"media_url"`    // optional low-res WebP image
    Order       int       `json:"order"`
    CreatedAt   time.Time `json:"created_at"`
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
- **Observations:** "Strong grasp of Boolean Algebra", "Studying consistently for 3 days".
- **Guidance:** "Continue Boolean Algebra" (Action: `/learning/act-2`).
