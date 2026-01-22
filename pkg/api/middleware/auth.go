package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// APIKeyAuth is a middleware that validates API keys and JWTs.
type APIKeyAuth struct {
	users     map[string]struct{} // Set of valid keys
	jwtSecret []byte
}

// NewAPIKeyAuth creates a new auth middleware.
func NewAPIKeyAuth(users []string, jwtSecret string) *APIKeyAuth {
	uMap := make(map[string]struct{})
	for _, k := range users {
		uMap[k] = struct{}{}
	}
	var secret []byte
	if jwtSecret != "" {
		secret = []byte(jwtSecret)
	}
	return &APIKeyAuth{users: uMap, jwtSecret: secret}
}

// Handler returns the middleware handler.
func (a *APIKeyAuth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip for health check and metrics
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" || r.URL.Path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}

		// 1. Check Authorization: Bearer <JWT> or <APIKey>
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Try to parse as JWT if enabled
			if a.jwtSecret != nil {
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return a.jwtSecret, nil
				})

				if err == nil && token.Valid {
					// Valid JWT!
					// TODO: Extract claims and set in context if needed
					next.ServeHTTP(w, r)
					return
				}
			}

			// If not JWT, try as API Key
			if _, ok := a.users[tokenString]; ok {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 2. Check X-API-Key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			if _, ok := a.users[apiKey]; ok {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
