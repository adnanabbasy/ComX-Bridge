package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/commatea/ComX-Bridge/pkg/core"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor handles gRPC authentication.
type GRPCAuthInterceptor struct {
	users     map[string]core.UserConfig // map[key]UserConfig
	jwtSecret []byte
}

// NewGRPCAuthInterceptor creates a new gRPC auth interceptor.
func NewGRPCAuthInterceptor(users []core.UserConfig, jwtSecret string) *GRPCAuthInterceptor {
	userMap := make(map[string]core.UserConfig)
	for _, u := range users {
		userMap[u.Key] = u
	}
	var secret []byte
	if jwtSecret != "" {
		secret = []byte(jwtSecret)
	}
	return &GRPCAuthInterceptor{users: userMap, jwtSecret: secret}
}

// authenticate checks the context for valid credentials.
func (i *GRPCAuthInterceptor) authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Check "authorization" (Bearer based) or "x-api-key"
	keys := md.Get("x-api-key")
	if len(keys) == 0 {
		// Try Authorization header
		auths := md.Get("authorization")
		if len(auths) > 0 {
			tokenString := auths[0]
			if strings.HasPrefix(tokenString, "Bearer ") {
				tokenString = strings.TrimPrefix(tokenString, "Bearer ")

				// Try as JWT
				if i.jwtSecret != nil {
					token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
						if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
						}
						return i.jwtSecret, nil
					})

					if err == nil && token.Valid {
						return nil // Valid JWT
					}
				}

				// If not JWT, treat as possible API Key (if passed as Bearer)
				keys = []string{tokenString}
			}
		}
	}

	if len(keys) == 0 {
		return status.Errorf(codes.Unauthenticated, "authentication required")
	}

	apiKey := keys[0]
	if _, ok := i.users[apiKey]; !ok {
		return status.Errorf(codes.Unauthenticated, "invalid api key")
	}

	return nil
}

// Unary returns a server interceptor for unary RPCs.
func (i *GRPCAuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := i.authenticate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// Stream returns a server interceptor for stream RPCs.
func (i *GRPCAuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := i.authenticate(ss.Context()); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
