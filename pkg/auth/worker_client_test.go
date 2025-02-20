package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateClusterRegistrationKey(t *testing.T) {
	tcs := []struct {
		key     string
		isError bool
	}{
		{
			key:     "clusterkey-1234567890",
			isError: false,
		},
		{
			key:     "default-cluster-registration-key-secret",
			isError: false,
		},
		{
			key:     "bogus",
			isError: true,
		},
		{
			key:     "",
			isError: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.key, func(t *testing.T) {
			err := os.Setenv(envVarName, tc.key)
			assert.NoError(t, err)
			err = ValidateClusterRegistrationKey()
			if tc.isError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
