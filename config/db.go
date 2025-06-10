package config

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var DB *sql.DB

func InitDB() {
	// 환경변수 로드
	host := os.Getenv("USERSERVICE_DB_HOST")
	port := os.Getenv("USERSERVICE_DB_PORT")
	user := os.Getenv("USERSERVICE_DB_USER")
	password := os.Getenv("USERSERVICE_DB_PASSWORD")
	name := os.Getenv("USERSERVICE_DB_NAME")

	DCS := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, name)

	var err error
	DB, err = sql.Open("postgres", DCS)
	if err != nil {
		panic("[DB 연결 안됨: " + err.Error() + "]")
	}

	// 연결 테스트
	log.Println("[DB 연결 테스트 중입니다..]")
	err = DB.Ping()
	if err != nil {
		panic("[DB 연결 테스트 실패: " + err.Error() + "]")
	}

	log.Println("[DB 연결이 성공적으로 완료되었습니다.]")

	checkTable(DB)
}

func checkTable(db *sql.DB) {
	query := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users');`

	var exists bool
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		log.Fatalf("[users 데이블 존재 여부 확인 실패 : %v]", err)
	}

	if !exists {
		log.Println("[users 테이블이 존재하지 않아 새로 생성합니다..]")
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
			log.Fatalf("[users 테이블 생성에 실패하였습니다. %v]", err)
		}
		log.Println("[users 테이블 생성에 성공하였습니다.]")
	} else {
		log.Println("[users 테이블이 이미 존재합니다.]")
	}
	// profiles 테이블 확인
	profileQuery := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'profiles');`
	err = db.QueryRow(profileQuery).Scan(&exists)
	if err != nil {
		log.Fatalf("[profiles 테이블 존재 여부 확인 실패 : %v]", err)
	}

	if !exists {
		log.Println("[profiles 테이블이 존재하지 않아 새로 생성합니다..]")
		createProfileQuery := `
        CREATE TABLE profiles (
                user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
                profile_image TEXT
                );`

		_, err := db.Exec(createProfileQuery)
		if err != nil {
			log.Fatalf("[profiles 테이블 생성에 실패하였습니다. %v]", err)
		}
		log.Println("[profiles 테이블 생성에 성공하였습니다.]")
	} else {
		log.Println("[profiles 테이블이 이미 존재합니다.]")
	}
}
