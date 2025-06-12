# NomoreGeobuk User Service

이 저장소는 Go로 작성된 간단한 RESTful 사용자 서비스입니다.
회원가입, 로그인, 프로필 관리 API를 제공합니다.

## 서버 실행 방법

```bash
# 의존성 설치
go mod download

# 빌드 및 실행
go run .
```

다음 환경 변수가 필요합니다.

- `USERSERVICE_DB_HOST`
- `USERSERVICE_DB_PORT`
- `USERSERVICE_DB_USER`
- `USERSERVICE_DB_PASSWORD`
- `USERSERVICE_DB_NAME`
- `JWT_SECRET`

서버는 `:8080` 포트에서 대기합니다.

## API 명세

### 회원가입

`POST /api/signup`

요청 본문(JSON):
```json
{
  "name": "string",
  "email": "string",
  "password": "string"
}
```

응답:
- `201 Created` 
  - ```json
    {
    "message": "회원가입이 성공적으로 완료되었습니다.",
    "result": "mail: test@example.com, name: 테스트"
    }
    ``` 

- `400 Bad Request` – 잘못된 입력 또는 비밀번호/이메일 형식 오류
- `401 Unauthorized` – 이미 사용 중인 이메일

### 로그인

`POST /api/signin`

요청 본문(JSON):
```json
{
  "email": "string",
  "password": "string"
}
```

응답:
- `200 OK`
  - ```json
    {
    "message": "로그인 성공 [test@example.com]",
    "result": {
        "name": "테스트",
        "token": "eyJhbGciOiJIUzI1NiasdAScxS6IkpXVCJ9.eyJleHAiOjE3NDk4MdsHsjMsInVzZXJfaWQiOiJmNjQ1NWM3Yi1lMjIzLTQ2MWYtOTAxYi0wZJhnKjkzMmY4MWIifQ.AcjpFSCwnFMcQQVfs4NMNRnMkDIItd_qOf42_XfviJA",
        "uuid": "f6455c7b-e223-461f-901b-0d2a6932f81b"
    }
}
    ``` 
- `400 Bad Request` – 잘못된 입력
- `401 Unauthorized` – 이메일 또는 비밀번호 오류

### 프로필

`POST /api/profile` – 프로필 저장 또는 수정

`GET /api/profile` – 현재 프로필 조회

두 엔드포인트 모두 로그인 시 발급받은 토큰을 `Authorization: Bearer <token>` 헤더로 전달해야 합니다.

#### POST 요청 본문(JSON)
```json
{
  "profile_image": "string"
}
```

응답:
- `200 OK` – 프로필 저장 완료
- `404 Not Found` – GET 요청 시 프로필이 존재하지 않음

## 테이블

애플리케이션 시작 시 테이블이 없으면 자동으로 생성됩니다.
- `users` – id, name, email, password hash 저장
- `profiles` – user id와 프로필 이미지 URL 저장
