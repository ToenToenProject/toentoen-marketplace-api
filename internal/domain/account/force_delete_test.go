package account

import (
	"fmt"
	"testing"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func generateTestOpsToken(role string, workerID string, accountID string) string {
	claims := jwt.MapClaims{
		"account_id": accountID,
		"worker_id":  workerID,
		"role":       role,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(config.Envs.JWTSecret))
	return tokenStr
}

func TestIsAllowedOperationsRole(t *testing.T) {
	allowed := []string{
		"cto", "CTO", " cto ",
		"admin", "ADMIN", "operations_admin", "super_admin", "superadmin",
		"cs", "CS", "admin/cs", "cs/admin", "customer_service", "customer service", "customer_support", "support",
	}

	for _, role := range allowed {
		assert.True(t, IsAllowedOperationsRole(role), fmt.Sprintf("Role '%s' should be allowed", role))
	}

	disallowed := []string{
		"customer", "merchant", "staff", "courier", "guest", "random", "",
	}

	for _, role := range disallowed {
		assert.False(t, IsAllowedOperationsRole(role), fmt.Sprintf("Role '%s' should be disallowed", role))
	}
}

func TestVerifyOperationsAuthorization(t *testing.T) {
	// Set mock config
	if config.Envs == nil {
		config.Envs = &config.AppConfig{}
	}
	config.Envs.AppEnv = "production"
	config.Envs.JWTSecret = "test-secret-key-32bytes-long12345"

	ctoUser := &models.OperationsUser{
		ID:       "OPS-001",
		WorkerID: "WRK-CTO-01",
		Name:     "Budi CTO",
		Role:     "cto",
		IsActive: true,
	}

	adminUser := &models.OperationsUser{
		ID:       "OPS-002",
		WorkerID: "WRK-ADM-01",
		Name:     "Siti Admin",
		Role:     "admin",
		IsActive: true,
	}

	csUser := &models.OperationsUser{
		ID:       "OPS-003",
		WorkerID: "WRK-CS-01",
		Name:     "Rian CS",
		Role:     "cs",
		IsActive: true,
	}

	unauthorizedUser := &models.OperationsUser{
		ID:       "OPS-004",
		WorkerID: "WRK-GUEST-01",
		Name:     "Guest User",
		Role:     "guest",
		IsActive: true,
	}

	repo := &mockProfileRepo{
		referralCodes: map[string]*models.ReferralUser{
			"WRK-CTO-01":   ctoUser,
			"WRK-ADM-01":   adminUser,
			"WRK-CS-01":    csUser,
			"WRK-GUEST-01": unauthorizedUser,
		},
	}

	service := NewProfileService(repo)

	t.Run("CTO via Bearer Token should succeed", func(t *testing.T) {
		token := generateTestOpsToken("cto", "WRK-CTO-01", "OPS-001")
		user, ok, err := service.VerifyOperationsAuthorization("Bearer "+token, "", "", "", "")
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "cto", user.Role)
	})

	t.Run("Admin via Bearer Token should succeed", func(t *testing.T) {
		token := generateTestOpsToken("admin", "WRK-ADM-01", "OPS-002")
		user, ok, err := service.VerifyOperationsAuthorization("Bearer "+token, "", "", "", "")
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "admin", user.Role)
	})

	t.Run("CS via Bearer Token should succeed", func(t *testing.T) {
		token := generateTestOpsToken("cs", "WRK-CS-01", "OPS-003")
		user, ok, err := service.VerifyOperationsAuthorization("Bearer "+token, "", "", "", "")
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "cs", user.Role)
	})

	t.Run("Unauthorized role in token should fail", func(t *testing.T) {
		token := generateTestOpsToken("courier", "WRK-GUEST-01", "OPS-004")
		_, ok, err := service.VerifyOperationsAuthorization("Bearer "+token, "", "", "", "")
		assert.False(t, ok)
		assert.Error(t, err)
	})

	t.Run("Authorized via X-Worker-ID in header", func(t *testing.T) {
		user, ok, err := service.VerifyOperationsAuthorization("", "", "WRK-ADM-01", "", "")
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "admin", user.Role)
	})

	t.Run("Authorized via X-Internal-Secret", func(t *testing.T) {
		user, ok, err := service.VerifyOperationsAuthorization("", "", "", "", config.Envs.JWTSecret)
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("Invalid secret and no token in production should fail", func(t *testing.T) {
		_, ok, err := service.VerifyOperationsAuthorization("", "", "", "", "wrong-secret")
		assert.False(t, ok)
		assert.Error(t, err)
	})
}
