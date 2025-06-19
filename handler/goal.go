package handler

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"user_auth/auth"
	"user_auth/config"
	"user_auth/model"
)

func createGoal(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Weekdays    []int    `json:"weekdays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "invalid payload", err.Error())
		log.Printf("[%s] %v", config.CallerName(1), err)
		return
	}
	if req.Name == "" {
		config.ResponseError(w, http.StatusBadRequest, "name required", "")
		return
	}

	tx, err := config.DB.Beginx()
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	defer tx.Rollback()

	var goalID int64
	if err := tx.QueryRow(`INSERT INTO goals (user_id, name, description) VALUES ($1,$2,$3) RETURNING goal_id`, uid, req.Name, req.Description).Scan(&goalID); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}

	for _, d := range req.Weekdays {
		if _, err := tx.Exec(`INSERT INTO goal_days (goal_id, weekday) VALUES ($1,$2)`, goalID, d); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
	}

	for _, tag := range req.Tags {
		var tagID int64
		if err := tx.QueryRow(`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING tag_id`, tag).Scan(&tagID); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
		if _, err := tx.Exec(`INSERT INTO goal_tags (goal_id, tag_id) VALUES ($1,$2)`, goalID, tagID); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
	}

	if err := tx.Commit(); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	config.ResponseOK(w, http.StatusCreated, "goal created", map[string]int64{"goal_id": goalID})
	log.Printf("[%s] 목표 생성 - %d", config.CallerName(1), goalID)
}

func listGoals(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)

	rows, err := config.DB.Queryx(`
        SELECT g.goal_id, g.name, COALESCE(g.description,'') AS description,
               COALESCE(string_agg(DISTINCT t.name, ','), '') AS tags,
               COALESCE(string_agg(DISTINCT gd.weekday::text, ','), '') AS weekdays
        FROM goals g
        LEFT JOIN goal_tags gt ON g.goal_id=gt.goal_id
        LEFT JOIN tags t ON gt.tag_id=t.tag_id
        LEFT JOIN goal_days gd ON g.goal_id=gd.goal_id
        WHERE g.user_id=$1
        GROUP BY g.goal_id
        ORDER BY g.created_at`, uid)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	defer rows.Close()

	var goals []model.Goal
	for rows.Next() {
		var (
			g       model.Goal
			tagsStr string
			daysStr string
		)
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &tagsStr, &daysStr); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
		if tagsStr != "" {
			g.Tags = strings.Split(tagsStr, ",")
		}
		if daysStr != "" {
			parts := strings.Split(daysStr, ",")
			for _, p := range parts {
				if v, err := strconv.Atoi(p); err == nil {
					g.Weekdays = append(g.Weekdays, v)
				}
			}
		}
		goals = append(goals, g)
	}

	config.ResponseOK(w, http.StatusOK, "goals", goals)
}

func updateGoal(w http.ResponseWriter, r *http.Request, id int64) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Weekdays    []int    `json:"weekdays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		config.ResponseError(w, http.StatusBadRequest, "invalid payload", err.Error())
		log.Printf("[%s] %v", config.CallerName(1), err)
		return
	}

	tx, err := config.DB.Beginx()
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE goals SET name=$1, description=$2, updated_at=now() WHERE goal_id=$3 AND user_id=$4`, req.Name, req.Description, id, uid)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		config.ResponseError(w, http.StatusNotFound, "goal not found", "")
		return
	}

	if _, err := tx.Exec(`DELETE FROM goal_days WHERE goal_id=$1`, id); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	for _, d := range req.Weekdays {
		if _, err := tx.Exec(`INSERT INTO goal_days (goal_id, weekday) VALUES ($1,$2)`, id, d); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
	}

	if _, err := tx.Exec(`DELETE FROM goal_tags WHERE goal_id=$1`, id); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	for _, tag := range req.Tags {
		var tagID int64
		if err := tx.QueryRow(`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING tag_id`, tag).Scan(&tagID); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
		if _, err := tx.Exec(`INSERT INTO goal_tags (goal_id, tag_id) VALUES ($1,$2)`, id, tagID); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
	}

	if err := tx.Commit(); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	config.ResponseOK(w, http.StatusOK, "goal updated", nil)
	log.Printf("[%s] 목표 수정 - %d", config.CallerName(1), id)
}

func deleteGoal(w http.ResponseWriter, r *http.Request, id int64) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)
	res, err := config.DB.Exec(`DELETE FROM goals WHERE goal_id=$1 AND user_id=$2`, id, uid)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	cnt, _ := res.RowsAffected()
	if cnt == 0 {
		config.ResponseError(w, http.StatusNotFound, "goal not found", "")
		return
	}
	config.ResponseOK(w, http.StatusOK, "goal deleted", nil)
	log.Printf("[%s] 목표 삭제 - %d", config.CallerName(1), id)
}

func completeActivity(w http.ResponseWriter, r *http.Request, id int64) {
	uid := r.Context().Value(auth.CtxUserID).(uuid.UUID)

	var goalUser uuid.UUID
	if err := config.DB.Get(&goalUser, `SELECT user_id FROM goals WHERE goal_id=$1`, id); err != nil {
		if err == sql.ErrNoRows {
			config.ResponseError(w, http.StatusNotFound, "goal not found", "")
		} else {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		}
		return
	}
	if goalUser != uid {
		config.ResponseError(w, http.StatusForbidden, "not owner", "")
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		config.ResponseError(w, http.StatusBadRequest, "multipart required", "")
		return
	}

	dateStr := r.FormValue("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	if dateStr != time.Now().Format("2006-01-02") {
		config.ResponseError(w, http.StatusBadRequest, "activity can only be completed today", "")
		return
	}
	note := r.FormValue("note")

	file, header, err := r.FormFile("image")
	if err != nil {
		config.ResponseError(w, http.StatusBadRequest, "image required", err.Error())
		return
	}
	defer file.Close()

	url, err := config.UploadToImgBB(file, header.Filename)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "upload error", err.Error())
		return
	}

	if _, err := config.DB.Exec(`INSERT INTO activities (goal_id, activity_date, image_url, note, completed_at) VALUES ($1,$2,$3,$4,now())`, id, dateStr, url, note); err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}

	config.ResponseOK(w, http.StatusOK, "activity completed", nil)
	log.Printf("[%s] 활동 완료 goal=%d", config.CallerName(1), id)
}

func listUserActivities(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	rows, err := config.DB.Queryx(`
                SELECT g.name, a.activity_date, a.image_url, COALESCE(a.note,'')
                FROM activities a
                JOIN goals g ON a.goal_id = g.goal_id
                WHERE g.user_id=$1 AND a.completed_at IS NOT NULL
                ORDER BY a.activity_date`, uid)
	if err != nil {
		config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
		return
	}
	defer rows.Close()

	type activity struct {
		Name  string `json:"name"`
		Date  string `json:"date"`
		Image string `json:"image"`
		Note  string `json:"note"`
	}
	var result []activity
	for rows.Next() {
		var a activity
		if err := rows.Scan(&a.Name, &a.Date, &a.Image, &a.Note); err != nil {
			config.ResponseError(w, http.StatusInternalServerError, "DB error", err.Error())
			return
		}
		result = append(result, a)
	}

	resp := map[string]interface{}{
		"id":     uid.String(),
		"result": result,
	}
	config.ResponseOK(w, http.StatusOK, "activities", resp)
	log.Printf("[%s] 활동 목록 조회 - %s", config.CallerName(1), uid)
}
