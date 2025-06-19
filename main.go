package main

import (
	"log"
	"net/http"
	"user_auth/auth"
	"user_auth/config"
	"user_auth/handler"
)

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)

	})
}

func main() {
	config.InitDB()

	mux := http.NewServeMux()

	// 공개 엔드포인트
	mux.HandleFunc("/api/signup", handler.SignUpHandler)
	mux.HandleFunc("/api/signin", handler.SignInHandler)
	mux.Handle("/api/goals", auth.Middleware(http.HandlerFunc(handler.GoalsHandler)))
	mux.Handle("/api/goals/", auth.Middleware(http.HandlerFunc(handler.GoalDetailHandler)))
	mux.Handle("/api/profile", auth.Middleware(http.HandlerFunc(handler.ProfileHandler)))
	mux.Handle("/api/activities", auth.Middleware(http.HandlerFunc(handler.UsersActivitiesHandler)))

	// CORS 래핑
	handlerWithCORS := enableCORS(mux)

	log.Printf("[%s] :8080 에서 작동중..\n---------서버 동작중---------", config.CallerName(1))
	if err := http.ListenAndServe(":8080", handlerWithCORS); err != nil {
		log.Fatal(err)
	}
}
