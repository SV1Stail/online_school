package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)
	require.NotContains(t, hash, "correct horse battery staple")

	valid, err := VerifyPassword("correct horse battery staple", hash)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = VerifyPassword("incorrect password", hash)
	require.NoError(t, err)
	require.False(t, valid)
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	require.Error(t, ValidatePassword("short"))
	require.NoError(t, ValidatePassword("long-enough-password"))
}
