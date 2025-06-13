package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"user_auth/auth"
	"user_auth/config"
	"user_auth/model"
)

// 회원가입 핸들러
func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		log.Printf("[%s] %v", config.CallerName(1), err)
		return
	}

	if config.IsValidEmail(req.Email) {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 이메일 형식", "")
		return
	}

	if !config.IsValidPassword(req.Password) {
		config.ResponseError(w, http.StatusBadRequest, "비밀번호는 6자리 이상 16자리 이하여야 합니다.", "")
		return
	}

	hash, err := config.HashPassword(req.Password)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "비밀번호 해쉬 오류", err.Error())
		return
	}

	_, err = config.DB.Exec(`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)`, req.Name, req.Email, hash)
	if err != nil {
		config.ResponseError(w, http.StatusUnauthorized, "중복되는 이메일", err.Error())
		return
	}

	// 문제 없을 시
	config.ResponseOK(w, http.StatusCreated, "회원가입이 성공적으로 완료되었습니다.", "mail: "+req.Email+", name: "+req.Name)
	log.Printf("[%s] 회원가입 성공 - %s", config.CallerName(1), req.Email)
}

// 로그인 핸들러
func SignInHandler(w http.ResponseWriter, r *http.Request) {

	// 0. OPTIONS 는 여기서 끊기 (프리플라이트)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		return
	}

	var user model.User
	row := config.DB.QueryRow(`SELECT user_id, name, password_hash FROM users WHERE email=$1`, req.Email)
	if err := row.Scan(&user.ID, &user.Name, &user.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 이메일", err.Error())
		} else {
			config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
		}
		return
	}

	if !config.CheckPasswordHash(req.Password, user.PasswordHash) {
		config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 비밀번호", "")
		return
	}

	token, err := config.GenerateJWT(user.ID)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "토큰 에러", err.Error())
		return
	}
	// 문제 없을 시
	log.Printf("[%s] 로그인 성공 - %v", config.CallerName(1), req.Email)
	config.ResponseOK(w, http.StatusOK, "로그인 성공 ["+req.Email+"]", map[string]string{
		"uuid":  user.ID,
		"name":  user.Name, //result.name
		"token": token,
	})
	// 프론트가 잘 못하겠다고 하면 아래 사용
	//config.ResponseOK(w, http.StatusOK, user.Name, token)
}

// 프로필 저장 및 조회 핸들러
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
