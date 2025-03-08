package okta

import (
	"testing"

	"github.com/golang-jwt/jwt"
	assert "github.com/stretchr/testify/assert"
)

func TestGetUserID(t *testing.T) {
	tcs := []struct {
		name    string
		claims  jwt.MapClaims
		want    string
		wantErr bool
	}{
		{
			name: "from uid",
			claims: jwt.MapClaims{
				"uid": "uid0",
				"sub": "sub0",
			},
			want: "uid0",
		},
		{
			name: "from sub",
			claims: jwt.MapClaims{
				"sub": "sub0",
			},
			want: "sub0",
		},
		{
			name: "none",
			claims: jwt.MapClaims{
				"foo": "bar",
			},
			wantErr: true,
		},
		{
			name: "non-string uid",
			claims: jwt.MapClaims{
				"sub": 0,
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getUserID(tc.claims)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
