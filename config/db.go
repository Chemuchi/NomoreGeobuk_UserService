package config

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"log"
	"os"
)

var DB *sqlx.DB

func InitDB() {
	// 환경변수 로드
	host := os.Getenv("USERSERVICE_DB_HOST")
	port := os.Getenv("USERSERVICE_DB_PORT")
	user := os.Getenv("USERSERVICE_DB_USER")
	password := os.Getenv("USERSERVICE_DB_PASSWORD")
	name := os.Getenv("USERSERVICE_DB_NAME")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, name)

	var err error
	DB, err = sqlx.Open("pgx", dsn)
	if err != nil {
		panic("[" + CallerName(1) + "] DB 연결 실패 : " + err.Error())
	}

	// 연결 테스트
	log.Printf("[%s] DB 연결 테스트중..", CallerName(1))
	err = DB.Ping()
	if err != nil {
		panic("[" + CallerName(1) + "] DB 연결 테스트 실패 : " + err.Error())
	}

	log.Printf("[%s] DB 연결됨", CallerName(1))
	checkTable(DB)
}

func checkTable(db *sqlx.DB) {
	query := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users');`

	var exists bool
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		log.Fatalf("[%s] users 테이블 존재여부 확인 실패: %v", CallerName(1), err.Error())
	}

	if !exists {
		log.Printf("[%s] users 테이블이 없어 생성중..", CallerName(1))
		createTableQuery := `
		CREATE TABLE users (
				user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(50) NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL, 
				created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
				);`

		_, err := db.Exec(createTableQuery)
		if err != nil {
			log.Fatalf("[%s] users 테이블 존재여부 확인 실패: %v", CallerName(1), err.Error())
		}
		log.Printf("[%s] users 테이블 생성 성공.", CallerName(1))
	} else {
		log.Printf("[%s] users 테이블 존재함.", CallerName(1))
	}
	// profiles 테이블 확인
	profileQuery := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'profiles');`
	err = db.QueryRow(profileQuery).Scan(&exists)
	if err != nil {
		log.Fatalf("[%s] profiles 테이블 존재여부 확인 실패: %v", CallerName(1), err.Error())
	}

	if !exists {
		log.Printf("[%s] profiles 테이블이 없어 생성중..")
		createProfileQuery := `
        CREATE TABLE profiles (
                user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
                profile_image TEXT
                );`

		_, err := db.Exec(createProfileQuery)
		if err != nil {
			log.Fatalf("[%s] profiles 테이블 생성에 실패함: %v", err.Error())
		}
		log.Printf("[%s] profiles 테이블 생성에 성공함.", CallerName(1))
	} else {
		log.Printf("[%s] profiles 테이블이 이미 존재함", CallerName(1))
	}
}
