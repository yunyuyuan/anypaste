package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret 在调用时读取，而不是包加载时——否则 main() 里加载 .env 之前
// 它就已经被求值成空了。
func jwtSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

const tokenTTL = 24 * 7 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
}

func IssueToken() (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "admin",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
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
