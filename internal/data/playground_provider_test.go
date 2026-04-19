package data

import (
	"context"
	"testing"

	"ruleGoKratos/internal/biz/entity"
	playgrounddata "ruleGoKratos/internal/data/playground"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewPlaygroundRuntimeRepoUsesGormImplementation(t *testing.T) {
	db := openProviderRuntimeTestDB(t)
	repo := NewPlaygroundRuntimeRepo(&Data{db: db})
	if repo == nil {
		t.Fatal("expected runtime repo")
	}
	gormRepo, ok := repo.(*playgrounddata.GormRuntimeRepo)
	if !ok {
		t.Fatalf("expected gorm runtime repo, got %T", repo)
	}
	plan := &entity.ExecutionPlan{PlanID: "plan-provider-runtime", PlanVersion: 1}
	if err := gormRepo.SavePlan(context.Background(), plan); err != nil {
		t.Fatalf("expected provider repo to auto-migrate runtime tables, got %v", err)
	}
}

func openProviderRuntimeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}
