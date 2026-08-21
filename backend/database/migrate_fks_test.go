package database

import (
	"path/filepath"
	"testing"
	"time"

	"log-backend/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func gormNow() time.Time { return time.Now() }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fk.db")+"?_foreign_keys=on&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Activity{},
		&domain.LearnerActivity{},
		&domain.Progress{},
		&domain.Observation{},
		&domain.Guidance{},
		&domain.DailyActivity{},
		&domain.Course{},
		&domain.Enrollment{},
		&domain.MicroModule{},
		&domain.Class{},
		&domain.ClassMember{},
		&domain.Assignment{},
		&domain.Submission{},
		&domain.AuditLog{},
		&domain.TokenBlocklist{},
		&domain.UserRevocation{},
		&domain.ConsentRecord{},
		&domain.ParentLink{},
		&domain.SupportIssue{},
		&domain.LearnerNote{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func fkCount(t *testing.T, db *gorm.DB, table string) int {
	t.Helper()
	var rows []struct {
		Table string `gorm:"column:table"`
	}
	if err := db.Raw("PRAGMA foreign_key_list(" + table + ")").Scan(&rows).Error; err != nil {
		t.Fatalf("foreign_key_list(%s): %v", table, err)
	}
	return len(rows)
}

// TestForeignKeysEnforced is the C4 proof: every declared constraint exists
// (PRAGMA foreign_key_list), and ON DELETE CASCADE actually removes learner
// rows when the parent user is deleted.
func TestForeignKeysEnforced(t *testing.T) {
	db := newTestDB(t)
	MigrateForeignKeys(db)

	for _, fc := range fkConstraints {
		if !hasForeignKey(db, fc.Table, fc.Column, fc.References) {
			t.Errorf("constraint missing: %s.%s -> %s.%s", fc.Table, fc.Column, fc.References, fc.RefColumn)
		}
	}

	// Cascade: delete the user, all learner rows must vanish with it.
	db.Create(&domain.User{ID: "cascade-user", Name: "Cascade Tester", Email: "cascade@test.edu", Role: domain.RoleStudent, IsVerified: true})
	db.Create(&domain.Activity{ID: "cascade-act", Title: "FK Test Activity", Description: "x", Topic: "Logic", Order: 99})
	db.Create(&domain.LearnerActivity{LearnerID: "cascade-user", ActivityID: "cascade-act", Status: "completed"})
	db.Create(&domain.Observation{ID: "cascade-obs", LearnerID: "cascade-user", Category: "strengths", Text: "x"})
	db.Create(&domain.Progress{LearnerID: "cascade-user", TotalTopics: 1, Completed: 1})
	db.Create(&domain.ParentLink{ID: "cascade-link", ParentID: "cascade-user", StudentID: "cascade-user", InviteCode: "FK-TEST-1", Status: "linked"})
	db.Create(&domain.SupportIssue{ID: "cascade-issue", UserID: "cascade-user", Category: "device", Description: "x", Status: "open"})
	db.Create(&domain.LearnerNote{ID: "cascade-note", StudentID: "cascade-user", TeacherID: "cascade-teacher", Note: "x"})
	db.Create(&domain.TokenBlocklist{JTI: "cascade-jti", UserID: "cascade-user", ExpiresAt: gormNow(), RevokedAt: gormNow()})

	if err := db.Unscoped().Delete(&domain.User{}, "id = ?", "cascade-user").Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}
	for _, probe := range []struct {
		model interface{}
		where string
	}{
		{&domain.LearnerActivity{}, "learner_id = ?"},
		{&domain.Progress{}, "learner_id = ?"},
		{&domain.Observation{}, "learner_id = ?"},
		{&domain.ParentLink{}, "parent_id = ? OR student_id = ?"},
		{&domain.SupportIssue{}, "user_id = ?"},
		{&domain.LearnerNote{}, "student_id = ?"},
		{&domain.TokenBlocklist{}, "user_id = ?"},
	} {
		var count int64
		db.Model(probe.model).Where(probe.where, "cascade-user", "cascade-user").Count(&count)
		if count != 0 {
			t.Errorf("cascade left %d rows for %T", count, probe.model)
		}
	}
}

// TestForeignKeysIdempotent proves the migration is safe to run repeatedly:
// the second pass finds every constraint already present and changes nothing.
func TestForeignKeysIdempotent(t *testing.T) {
	db := newTestDB(t)
	MigrateForeignKeys(db)
	before := 0
	for _, fc := range fkConstraints {
		before += fkCount(t, db, fc.Table)
	}
	MigrateForeignKeys(db)
	after := 0
	for _, fc := range fkConstraints {
		after += fkCount(t, db, fc.Table)
	}
	if before != after {
		t.Fatalf("idempotency broken: %d constraints before, %d after", before, after)
	}
}

// TestForeignKeyAnonymizedColumnsStayUnconstrained pins the C4 safety
// contract: columns the erasure map anonymizes to "" (or optional refs) must
// NEVER carry a FK — a constraint would reject the blanking.
func TestForeignKeyAnonymizedColumnsStayUnconstrained(t *testing.T) {
	db := newTestDB(t)
	MigrateForeignKeys(db)

	for _, table := range []string{"audit_logs", "announcements", "assignments", "classes"} {
		if n := fkCount(t, db, table); n != 0 {
			t.Errorf("table %s must have no FKs (anonymized refs), got %d", table, n)
		}
	}
}

// TestForeignKeyOrphanSkip proves the migration is non-destructive: a table
// with rows referencing a missing parent is skipped with data preserved, not
// deleted to fit the constraint.
func TestForeignKeyOrphanSkip(t *testing.T) {
	db := newTestDB(t)
	db.Create(&domain.LearnerActivity{LearnerID: "ghost-learner", ActivityID: "ghost-activity", Status: "completed"})
	MigrateForeignKeys(db)

	if hasForeignKey(db, "learner_activities", "learner_id", "users") {
		t.Fatalf("constraint must not be created over orphan rows")
	}
	var count int64
	db.Model(&domain.LearnerActivity{}).Count(&count)
	if count != 1 {
		t.Fatalf("orphan row must be preserved, got %d rows", count)
	}
}
