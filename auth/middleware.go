package auth

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"user_auth/config"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			config.ResponseError(w, http.StatusUnauthorized, "토큰 없음", "")
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			config.ResponseError(w, http.StatusUnauthorized, "토큰 검증 실패", err.Error())
			return
		}

		uidStr, ok := claims["user_id"].(string)
		if !ok {
			config.ResponseError(w, http.StatusUnauthorized, "user_id 누락", "")
			return
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			config.ResponseError(w, http.StatusUnauthorized, "user_id 형식 오류", err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), CtxUserID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
