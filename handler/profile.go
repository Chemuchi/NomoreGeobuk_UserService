package handler

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"user_auth/auth"
	"user_auth/config"
)

func GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.CtxUserID).(uuid.UUID)

	var profile struct {
		Name         string `db:"name" json:"name"`
		Email        string `db:"email" json:"email"`
		ProfileImage string `db:"profile_image" json:"profile_image"`
	}

	if err := config.DB.Get(&profile, `
        SELECT u.name, u.email, COALESCE(p.profile_image,'') AS profile_image
        FROM users u
        LEFT JOIN profiles p ON p.user_id = u.user_id
        WHERE u.user_id=$1`, userID); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
		return
	}

	config.ResponseOK(w, http.StatusOK, "프로필 조회 성공", profile)

}

func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	uidVal := r.Context().Value(auth.CtxUserID)
	userID, ok := uidVal.(uuid.UUID)
	if !ok {
		config.ResponseError(w, http.StatusUnauthorized, "토큰이 없거나 잘못됨", "")
		return
	}
	// 1) multipart/form-data 파일 업로드 플로우
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		file, header, err := r.FormFile("image")
		if err != nil {
			config.ResponseError(w, http.StatusBadRequest, "이미지 파일 없음", err.Error())
			return
		}
		defer file.Close()

		// 새로운 파일명 생성: avatar-(user_id)-(랜덤값).확장자
		ext := filepath.Ext(header.Filename)
		rand.Seed(time.Now().UnixNano())
		randPart := rand.Int63()
		newName := fmt.Sprintf("avatar-%s-%d%s", userID.String(), randPart, ext)

		url, err := config.UploadToImgBB(file, newName)
		if err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "ImgBB 업로드 실패", err.Error())
			return
		}

		if _, err := config.DB.Exec(`
            INSERT INTO profiles (user_id, profile_image)
            VALUES ($1, $2)
            ON CONFLICT (user_id) DO UPDATE SET profile_image = EXCLUDED.profile_image`,
			userID, url); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB 업데이트 실패", err.Error())
			return
		}

		config.ResponseOK(w, http.StatusOK, "프로필 이미지 업로드 성공", map[string]string{"profile_image": url})
		log.Printf("[%s] 프로필 이미지 업데이트함 - %s", config.CallerName(1), userID)
		return
	}

	var payload struct {
		ProfileImage string `json:"profile_image"`
	}
	if decodeErr := json.NewDecoder(r.Body).Decode(&payload); decodeErr != nil || payload.ProfileImage == "" {
		config.ResponseError(w, http.StatusBadRequest, "invalid payload", "")
		return
	}

	if _, err := config.DB.Exec(`
        INSERT INTO profiles (user_id, profile_image)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE SET profile_image = EXCLUDED.profile_image`,
		userID, payload.ProfileImage); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB 업데이트 실패", err.Error())
		return
	}

	config.ResponseOK(w, http.StatusOK, "프로필 이미지 URL 업데이트 성공", map[string]string{"profile_image": payload.ProfileImage})
	log.Printf("[%s] 프로필 이미지 업데이트함 - %s", config.CallerName(1), userID)
}
