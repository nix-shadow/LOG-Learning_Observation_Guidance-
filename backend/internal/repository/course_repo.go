package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

// randomID mirrors the service-layer ID generator (auth_utils) without
// forcing a repository->service import. FirstOrCreate ignores it when the
// row already exists, so a wasted random value is harmless.
func randomID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

type courseRepo struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) domain.CourseRepository {
	return &courseRepo{db: db}
}

func (r *courseRepo) FindCourses(ctx context.Context, userID string, page, limit int) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	r.db.WithContext(ctx).Model(&domain.Course{}).Count(&total)

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	if len(courses) == 0 {
		return courses, total, nil
	}

	// WP-0.2 C5: derive both numbers from the enrollments table. The static
	// Enrolled column on Course was seed-only and fabricated — real counts
	// come from real rows.
	type countRow struct {
		CourseID string
		Count    int
	}
	var counts []countRow
	if err := r.db.WithContext(ctx).
		Model(&domain.Enrollment{}).
		Select("course_id, COUNT(*) AS count").
		Group("course_id").Scan(&counts).Error; err != nil {
		return nil, 0, err
	}
	countByCourse := make(map[string]int, len(counts))
	for _, c := range counts {
		countByCourse[c.CourseID] = c.Count
	}

	var myRows []domain.Enrollment
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&myRows).Error; err != nil {
		return nil, 0, err
	}
	mySet := make(map[string]bool, len(myRows))
	for _, e := range myRows {
		mySet[e.CourseID] = true
	}

	for i := range courses {
		courses[i].Enrolled = countByCourse[courses[i].ID]
		courses[i].IsEnrolled = mySet[courses[i].ID]
	}
	return courses, total, nil
}

func (r *courseRepo) Enroll(ctx context.Context, userID, courseID string) error {
	var course domain.Course
	if err := r.db.WithContext(ctx).First(&course, "id = ?", courseID).Error; err != nil {
		return err // gorm.ErrRecordNotFound propagates; service maps it to 404
	}
	// First, then Create (FirstOrCreate misbehaves with a preset string PK).
	var existing domain.Enrollment
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&existing).Error
	if err == nil {
		return nil // already enrolled — idempotent
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	err = r.db.WithContext(ctx).Create(&domain.Enrollment{
		ID:       randomID("enr"),
		UserID:   userID,
		CourseID: courseID,
	}).Error
	if err != nil {
		// Unique-index race (two tabs enrolling at once): the row now exists.
		var dup int64
		r.db.WithContext(ctx).Model(&domain.Enrollment{}).
			Where("user_id = ? AND course_id = ?", userID, courseID).Count(&dup)
		if dup > 0 {
			return nil
		}
		return err
	}
	return nil
}

// Unenroll is idempotent: removing an enrollment that does not exist is not
// an error — the desired state is reached either way.
func (r *courseRepo) Unenroll(ctx context.Context, userID, courseID string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).
		Delete(&domain.Enrollment{}).Error
}

func (r *courseRepo) FindModulesByActivityID(ctx context.Context, activityID string) ([]domain.MicroModule, error) {
	var modules []domain.MicroModule
	err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("`order` asc").Find(&modules).Error
	return modules, err
}