package config

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	nameCache = map[uintptr]string{}
	mu        sync.RWMutex
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// 회원가입 관련 로직
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidPassword(password string) bool {
	return len(password) >= 6 && len(password) <= 16
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// 로그인 관련 로직
func GenerateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func ParseJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if uid, ok := claims["user_id"].(string); ok {
			return uid, nil
		}
	}
	return "", fmt.Errorf("invalid token")
}

func CallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	mu.RLock()
	if v, ok := nameCache[pc]; ok {
		mu.RUnlock()
		return v
	}
	mu.RUnlock()

	fn := runtime.FuncForPC(pc)
	full := fn.Name()

	if idx := strings.LastIndex(full, "/"); idx != -1 {
		full = full[idx+1:]
	}

	name := strings.Replace(full, ".", ":", 1) // 패키지명:함수명

	mu.Lock()
	nameCache[pc] = name
	mu.Unlock()
	return name
}

// 응답 관련 로직
func ResponseError(w http.ResponseWriter, status int, message string, error string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
		"error":   error,
	})
}

func ResponseOK(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"result":  result,
	})
}
