package handler

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
	"user_auth/auth"
	"user_auth/config"
	"user_auth/model"
)

func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
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

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetProfile(w, r)
	case http.MethodPost:
		UpdateProfile(w, r)
	default:
		http.Error(w, "허용되지 않는 메소드", http.StatusMethodNotAllowed)
	}
}

func UsersActivitiesHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)
	switch r.Method {
	case http.MethodGet:
		listUserActivities(w, r, uid)
	default:
		http.Error(w, "허용되지 않는 메소드", http.StatusMethodNotAllowed)
	}
}

// GoalsHandler handles /api/goals for listing and creating goals
func GoalsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listGoals(w, r)
	case http.MethodPost:
		createGoal(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GoalDetailHandler handles /api/goals/{id} and /api/goals/{id}/activities
func GoalDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/goals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	goalID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		config.ResponseError(w, http.StatusBadRequest, "invalid goal id", err.Error())
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			updateGoal(w, r, goalID)
		case http.MethodDelete:
			deleteGoal(w, r, goalID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "activities" && r.Method == http.MethodPost {
		completeActivity(w, r, goalID)
		return
	}
	http.NotFound(w, r)
}
