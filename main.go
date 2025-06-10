package main

import (
	"log"
	"net/http"
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

	mux.HandleFunc("/api/signup", handler.SignUpHandler)
	mux.HandleFunc("/api/signin", handler.SignInHandler)
	mux.HandleFunc("/api/profile", handler.ProfileHandler)

	if err := http.ListenAndServe(":8080", enableCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
