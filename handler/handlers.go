package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	log.Println("[signup 요청 들어옴]")
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		log.Println("[signup 요청 실패: 유효하지 않는 입력]")
		return
	}

	if !config.IsValidEmail(req.Email) {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 이메일 형식", "")
		log.Println("[signup 요청 실패: 유효하지 않는 이메일 형식]")
		return
	}

	if !config.IsValidPassword(req.Password) {
		config.ResponseError(w, http.StatusBadRequest, "비밀번호는 6자리 이상 16자리 이하여야 합니다.", "")
		log.Println("[signup 요청 실패: 비밀번호는 6자리 이상 16자리 이하여야 함]")
		return
	}

	hash, err := config.HashPassword(req.Password)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "비밀번호 해쉬 오류", err.Error())
		log.Println("[signup 요청 실패: 비밀번호 해쉬 오류]")
		return
	}

	_, err = config.DB.Exec(`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)`, req.Name, req.Email, hash)
	if err != nil {
		config.ResponseError(w, http.StatusUnauthorized, "중복되는 이메일", err.Error())
		log.Println("[signup 요청 실패: 중복된 이메일]")
		return
	}

	// 문제 없을 시
	config.ResponseOK(w, http.StatusCreated, "회원가입이 성공적으로 완료되었습니다.", "mail: "+req.Email+", name: "+req.Name)
	log.Printf("[signup 요청 정상 작동 완료. 메일: %s]\n", req.Email)
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
	log.Println("[signin 요청 들어옴]")
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		log.Println("[signin 요청 실패: 유효하지 않는 입력]")
		return
	}

	var user model.User
	row := config.DB.QueryRow(`SELECT user_id, password_hash FROM users WHERE email=$1`, req.Email)
	if err := row.Scan(&user.ID, &user.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 이메일", "")
			log.Println("[signin 요청 실패: 유효하지 않는 이메일]")
		} else {
			config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
			log.Println("[signin 요청 실패: DB 오류]")
		}
		return
	}

	if !config.CheckPasswordHash(req.Password, user.PasswordHash) {
		config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 비밀번호", "")
		log.Println("[signin 요청 실패: 유효하지 않는 비밀번호]")
		return
	}

	token, err := config.GenerateJWT(user.ID)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "토큰 에러", err.Error())
		log.Println("[signin 요청 실패: 토큰 오류]")
		return
	}
	// 문제 없을 시
	log.Println("[signin 요청 정상 작동 완료. 메일 : " + req.Email + "]")
	config.ResponseOK(w, http.StatusOK, "로그인 성공 ["+req.Email+"]", token)
}

// 프로필 저장 및 조회 핸들러
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		config.ResponseError(w, http.StatusUnauthorized, "토큰이 필요합니다.", "")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	userID, err := config.ParseJWT(tokenString)
	if err != nil {
		config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 토큰", err.Error())
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			ProfileImage string `json:"profile_image"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			config.ResponseError(w, http.StatusBadRequest, "유효하지 않은 입력", err.Error())
			return
		}
		_, err = config.DB.Exec(`
        INSERT INTO profiles (user_id, profile_image)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id) DO UPDATE SET profile_image = EXCLUDED.profile_image`,
			userID, req.ProfileImage)
		if err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
			return
		}
		config.ResponseOK(w, http.StatusOK, "프로필 저장 완료", "")
	case http.MethodGet:
		var profile model.Profile
		row := config.DB.QueryRow(`SELECT user_id, profile_image FROM profiles WHERE user_id=$1`, userID)
		if err := row.Scan(&profile.UserID, &profile.ProfileImage); err != nil {
			if err == sql.ErrNoRows {
				config.ResponseError(w, http.StatusNotFound, "프로필이 없습니다.", "")
			} else {
				config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
