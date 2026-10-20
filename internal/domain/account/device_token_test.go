package account

import (
	"errors"
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockDeviceTokenRepo struct {
	mockProfileRepo
	lastUpsertAccountID string
	lastUpsertToken     string
	lastUpsertPlatform  string
	lastDeleteAccountID string
	lastDeleteToken     string
	failUpsert          bool
	failDelete          bool
}

func (m *mockDeviceTokenRepo) UpsertDeviceToken(accountID string, token string, platform string) error {
	if m.failUpsert {
		return errors.New("db error")
	}
	m.lastUpsertAccountID = accountID
	m.lastUpsertToken = token
	m.lastUpsertPlatform = platform
	return nil
}

func (m *mockDeviceTokenRepo) DeleteDeviceToken(accountID string, token string) error {
	if m.failDelete {
		return errors.New("db error")
	}
	m.lastDeleteAccountID = accountID
	m.lastDeleteToken = token
	return nil
}

func TestRegisterDeviceToken(t *testing.T) {
	t.Run("Success Register Device Token with Android", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeviceTokenInput{
			DeviceToken: "fcm_token_12345",
			Platform:    "android",
		}
		err := svc.RegisterDeviceToken("ACC-001", input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastUpsertAccountID != "ACC-001" {
			t.Errorf("expected account ID ACC-001, got %s", repo.lastUpsertAccountID)
		}
		if repo.lastUpsertToken != "fcm_token_12345" {
			t.Errorf("expected token fcm_token_12345, got %s", repo.lastUpsertToken)
		}
		if repo.lastUpsertPlatform != "android" {
			t.Errorf("expected platform android, got %s", repo.lastUpsertPlatform)
		}
	})

	t.Run("Success Register with iOS and Default Platform Fallback", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeviceTokenInput{
			DeviceToken: "apns_fcm_token_67890",
			Platform:    "IOS",
		}
		err := svc.RegisterDeviceToken("ACC-002", input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastUpsertPlatform != "ios" {
			t.Errorf("expected normalized platform ios, got %s", repo.lastUpsertPlatform)
		}

		// Empty platform should default to android
		input2 := models.DeviceTokenInput{
			DeviceToken: "token_no_platform",
		}
		err = svc.RegisterDeviceToken("ACC-002", input2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastUpsertPlatform != "android" {
			t.Errorf("expected default platform android, got %s", repo.lastUpsertPlatform)
		}
	})

	t.Run("Validation: Missing Device Token", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeviceTokenInput{
			Platform: "android",
		}
		err := svc.RegisterDeviceToken("ACC-001", input)
		if err == nil {
			t.Errorf("expected error when device_token is empty, got nil")
		}
	})

	t.Run("Validation: Missing Account ID", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeviceTokenInput{
			DeviceToken: "valid_token",
			Platform:    "android",
		}
		err := svc.RegisterDeviceToken("", input)
		if err == nil {
			t.Errorf("expected error when account ID is empty, got nil")
		}
	})
}

func TestDeleteDeviceToken(t *testing.T) {
	t.Run("Success Delete Specific Token", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeleteDeviceTokenInput{
			DeviceToken: "fcm_token_to_remove",
		}
		err := svc.DeleteDeviceToken("ACC-001", input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastDeleteAccountID != "ACC-001" {
			t.Errorf("expected account ID ACC-001, got %s", repo.lastDeleteAccountID)
		}
		if repo.lastDeleteToken != "fcm_token_to_remove" {
			t.Errorf("expected token fcm_token_to_remove, got %s", repo.lastDeleteToken)
		}
	})

	t.Run("Success Delete All Tokens on Logout (Empty Payload)", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeleteDeviceTokenInput{}
		err := svc.DeleteDeviceToken("ACC-001", input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastDeleteAccountID != "ACC-001" {
			t.Errorf("expected account ID ACC-001, got %s", repo.lastDeleteAccountID)
		}
		if repo.lastDeleteToken != "" {
			t.Errorf("expected empty token for all delete, got %s", repo.lastDeleteToken)
		}
	})

	t.Run("Validation: Missing Account ID", func(t *testing.T) {
		repo := &mockDeviceTokenRepo{}
		svc := NewProfileService(repo)

		input := models.DeleteDeviceTokenInput{DeviceToken: "abc"}
		err := svc.DeleteDeviceToken("  ", input)
		if err == nil {
			t.Errorf("expected error when account ID is empty, got nil")
		}
	})
}
