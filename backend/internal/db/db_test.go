package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitDBMigratesAndSeeds(t *testing.T) {
	path := t.TempDir() + "/qp-test.db"
	err := InitDB(Config{
		Path:          path,
		AdminEmail:    "admin@example.com",
		AdminPassword: "secret123",
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = DB.Close() })

	// Seed admin exists with admin role.
	var u UserEntity
	require.NoError(t, Gorm.Where("email = ?", "admin@example.com").First(&u).Error)
	assert.Equal(t, "admin", u.Role)
	assert.True(t, u.IsActive)
	assert.NotEmpty(t, u.HashedPassword)

	// Re-running InitDB is idempotent (no duplicate admin).
	require.NoError(t, InitDB(Config{
		Path:          path,
		AdminEmail:    "admin@example.com",
		AdminPassword: "secret123",
	}, zap.NewNop()))
	var count int64
	require.NoError(t, Gorm.Model(&UserEntity{}).Where("email = ?", "admin@example.com").Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// The raw host_metrics table exists and is queryable.
	var n int64
	require.NoError(t, Gorm.Raw("SELECT COUNT(*) FROM host_metrics").Scan(&n).Error)
	assert.Equal(t, int64(0), n)
}
