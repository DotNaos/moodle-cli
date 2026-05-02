package chatgptapp

import (
	"net/http"

	"github.com/DotNaos/moodle-services/internal/auth"
)

var ErrUnauthorized = auth.ErrUnauthorized

func APIKeyFromRequest(r *http.Request) string {
	return auth.APIKeyFromRequest(r)
}

func HashAPIKey(key string) string {
	return auth.HashAPIKey(key)
}

func ConstantTimeEqual(left string, right string) bool {
	return auth.ConstantTimeEqual(left, right)
}
