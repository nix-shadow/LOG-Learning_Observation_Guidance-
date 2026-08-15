package repository

import (
	"context"

	"log-backend/internal/domain"

	"gorm.io/gorm"
)

type courseRepo struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) domain.CourseRepository {
	return &courseRepo{db: db}
}

func (r *courseRepo) FindCourses(ctx context.Context, page, limit int) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	r.db.WithContext(ctx).Model(&domain.Course{}).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&courses).Error

	return courses, total, err
}

func (r *courseRepo) FindModulesByActivityID(ctx context.Context, activityID string) ([]domain.MicroModule, error) {
	var modules []domain.MicroModule
	err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("`order` asc").Find(&modules).Error
	return modules, err
}
