package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"yunyuyuan/anypaste/internal/config"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
)

// store holds the JWT secret and admin password hash. main() injects it via
// UseStore before the server starts serving.
var store *config.Store

// UseStore wires the config store the auth package reads its secrets from.
func UseStore(s *config.Store) { store = s }

func jwtSecret() []byte {
	return []byte(store.JwtSecret())
}

// DefaultTokenTTL is the session lifetime used when a caller passes ttl <= 0
// (e.g. the web login). It also doubles as the upper bound callers should clamp
// any client-requested TTL to.
const DefaultTokenTTL = 24 * 7 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
}

// IssueToken signs a session token valid for ttl; ttl <= 0 falls back to DefaultTokenTTL.
func IssueToken(ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "admin",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "owner",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func PasteToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// bearerToken 从 "Bearer <token>" 头里取出 token，容忍前缀后有无空格。
func bearerToken(header string) string {
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
}

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := bearerToken(r.Header.Get("Authorization"))
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		if _, err := PasteToken(tokenStr); err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewAuthUnaryInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
			tokenStr := bearerToken(ar.Header().Get("Authorization"))
			if tokenStr == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing token"))
			}
			if _, err := PasteToken(tokenStr); err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}
			return next(ctx, ar)
		}
	}
}
