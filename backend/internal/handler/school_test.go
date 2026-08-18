package handler

import (
	"context"
	"testing"

	"log-backend/database"
	"log-backend/internal/domain"
	"log-backend/internal/repository"
	"log-backend/internal/service"
)

func newSchoolTestHandler(t *testing.T) *SchoolHandler {
	repo := repository.NewSchoolRepository(database.DB)
	return NewSchoolHandler(service.NewSchoolService(repo))
}

func newTestUser(t *testing.T, role domain.Role) *domain.User {
	user := &domain.User{
		ID:         service.GenerateSecureID("user"),
		Email:      service.GenerateSecureID("u") + "@test.local",
		Name:       "Test " + string(role),
		Role:       role,
		IsVerified: true,
	}
	if err := repository.NewUserRepository(database.DB).Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Unscoped().Where("id = ?", user.ID).Delete(&domain.User{})
	})
	return user
}
