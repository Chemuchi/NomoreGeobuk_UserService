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

	// 추가 테이블 및 트리거 생성
	if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm;`); err != nil {
		log.Fatalf("[%s] pg_trgm extension 생성 실패: %v", CallerName(1), err.Error())
	} else {
		log.Printf("[%s] pg_trgm extension 확인 완료", CallerName(1))
	}

	ensureTable := func(name, query string) {
		var exists bool
		err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, name).Scan(&exists)
		if err != nil {
			log.Fatalf("[%s] %s 테이블 존재여부 확인 실패: %v", CallerName(1), name, err.Error())
		}
		if !exists {
			log.Printf("[%s] %s 테이블이 없어 생성중..", CallerName(1), name)
			if _, err := db.Exec(query); err != nil {
				log.Fatalf("[%s] %s 테이블 생성 실패: %v", CallerName(1), name, err.Error())
			}
			log.Printf("[%s] %s 테이블 생성 성공", CallerName(1), name)
		} else {
			log.Printf("[%s] %s 테이블이 이미 존재함", CallerName(1), name)
		}
	}

	ensureTable("goals", `CREATE TABLE goals (
                goal_id BIGSERIAL PRIMARY KEY,
                user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
                name VARCHAR(100) NOT NULL,
                description TEXT,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )`)

	ensureTable("tags", `CREATE TABLE tags (
                tag_id BIGSERIAL PRIMARY KEY,
                name VARCHAR(30) UNIQUE NOT NULL
        )`)

	ensureTable("goal_tags", `CREATE TABLE goal_tags (
                goal_id BIGINT NOT NULL REFERENCES goals(goal_id) ON DELETE CASCADE,
                tag_id  BIGINT NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
                PRIMARY KEY (goal_id, tag_id)
        )`)

	if _, err := db.Exec(`
        CREATE OR REPLACE FUNCTION fn_limit_5_tags() RETURNS trigger LANGUAGE plpgsql AS $$
        BEGIN
                IF (SELECT count(*) FROM goal_tags WHERE goal_id = NEW.goal_id) >= 5 THEN
                        RAISE EXCEPTION 'A goal can have at most 5 tags';
                END IF;
                RETURN NEW;
        END $$;
    `); err != nil {
		log.Fatalf("[%s] fn_limit_5_tags 생성 실패: %v", CallerName(1), err.Error())
	}

	if _, err := db.Exec(`DROP TRIGGER IF EXISTS trg_limit_5_tags ON goal_tags; CREATE TRIGGER trg_limit_5_tags BEFORE INSERT ON goal_tags FOR EACH ROW EXECUTE FUNCTION fn_limit_5_tags();`); err != nil {
		log.Fatalf("[%s] trg_limit_5_tags 생성 실패: %v", CallerName(1), err.Error())
	} else {
		log.Printf("[%s] trg_limit_5_tags 확인 완료", CallerName(1))
	}

	ensureTable("goal_days", `CREATE TABLE goal_days (
                goal_id BIGINT NOT NULL REFERENCES goals(goal_id) ON DELETE CASCADE,
                weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
                PRIMARY KEY (goal_id, weekday)
        )`)

	ensureTable("activities", `CREATE TABLE activities (
                activity_id   BIGSERIAL PRIMARY KEY,
                goal_id       BIGINT NOT NULL REFERENCES goals(goal_id) ON DELETE CASCADE,
                activity_date DATE NOT NULL,
                image_url     TEXT NOT NULL,
                note          TEXT,
                completed_at  TIMESTAMPTZ,
                UNIQUE (goal_id, activity_date)
        )`)

	if _, err := db.Exec(`
        CREATE OR REPLACE FUNCTION fn_check_day_match() RETURNS trigger LANGUAGE plpgsql AS $$
        DECLARE
                w SMALLINT;
        BEGIN
                w := EXTRACT(DOW FROM NEW.activity_date);
                IF w = 0 THEN w := 7; END IF;

                IF NOT EXISTS (
                        SELECT 1 FROM goal_days WHERE goal_id = NEW.goal_id AND weekday = w
                ) THEN
                        RAISE EXCEPTION 'Goal % is not scheduled on weekday %', NEW.goal_id, w;
                END IF;

                RETURN NEW;
        END $$;
    `); err != nil {
		log.Fatalf("[%s] fn_check_day_match 생성 실패: %v", CallerName(1), err.Error())
	}

	if _, err := db.Exec(`DROP TRIGGER IF EXISTS trg_check_day_match ON activities; CREATE TRIGGER trg_check_day_match BEFORE INSERT ON activities FOR EACH ROW EXECUTE FUNCTION fn_check_day_match();`); err != nil {
		log.Fatalf("[%s] trg_check_day_match 생성 실패: %v", CallerName(1), err.Error())
	} else {
		log.Printf("[%s] trg_check_day_match 확인 완료", CallerName(1))
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_goals_name_trgm ON goals USING gin (name gin_trgm_ops);`); err != nil {
		log.Fatalf("[%s] goals 인덱스 생성 실패: %v", CallerName(1), err.Error())
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tags_name_trgm ON tags USING gin (name gin_trgm_ops);`); err != nil {
		log.Fatalf("[%s] tags 인덱스 생성 실패: %v", CallerName(1), err.Error())
	}
}
