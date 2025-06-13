package auth

// context 키를 string 대신 전용 타입으로 만들면 오타·충돌 방지
type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
)
