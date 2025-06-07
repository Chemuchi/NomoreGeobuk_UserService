package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"user_auth/config"
	"user_auth/model"
)

// 회원가입 핸들러
func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[signup 요청 들어옴]")
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		return
	}

	if !config.IsValidEmail(req.Email) {
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
		//http.Error(w, "해쉬 오류", http.StatusInternalServerError)
		return
	}

	_, err = config.DB.Exec(`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)`, req.Name, req.Email, hash)
	if err != nil {
		config.ResponseError(w, http.StatusUnauthorized, "중복되는 이메일", err.Error())
		return
	}

	// 문제 없을 시
	config.ResponseOK(w, http.StatusCreated, "회원가입이 성공적으로 완료되었습니다.", "mail: "+req.Email+", name: "+req.Name)
	log.Printf("[signup 요청 정상 작동함. 메일: %s]", req.Email)
}

// 로그인 핸들러
func SignInHandler(w http.ResponseWriter, r *http.Request) {

	// SignInHandler 맨 위
	log.Printf("Content-Length: %d", r.ContentLength)
	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("Raw Body: %s", string(bodyBytes))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // 다시 읽을 수 있게 복구

	// 0. OPTIONS 는 여기서 끊기 (프리플라이트)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.Println("[signin 요청 들어옴]")
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// 이하 예외 처리
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "유효하지 않는 입력", err.Error())
		//http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var user model.User
	row := config.DB.QueryRow(`SELECT id, password_hash FROM users WHERE email=$1`, req.Email)
	if err := row.Scan(&user.ID, &user.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 이메일", "")
		} else {
			config.ResponseError(w, http.StatusInternalServerError, "DB 오류", err.Error())
		}
		return
	}

	if !config.CheckPasswordHash(req.Password, user.PasswordHash) {
		config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 비밀번호", "")
		return
	}
	/*var exists int
	err := config.DB.QueryRow(`SELECT 1 FROM users WHERE email=$1`, req.Email).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			config.ResponseError(w, http.StatusUnauthorized, "유효하지 않은 이메일임", "이메일이 존재하지 않습니다.")
		} else {
			config.ResponseError(w, http.StatusInternalServerError, "서버 오류", err.Error())
		}
		return
	}

	if !config.CheckPasswordHash(req.Password, user.PasswordHash) {
		config.ResponseError(w, http.StatusUnauthorized, "유효하지 않는 비밀번호임", "")
		//http.Error(w, "Invalid Password", http.StatusInternalServerError)
		return
	}*/

	token, err := config.GenerateJWT(user.ID)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "토큰 에러", err.Error())
		return
	}
	// 문제 없을 시
	config.ResponseOK(w, http.StatusOK, "로그인 성공", token)
}
