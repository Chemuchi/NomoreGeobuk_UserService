# NomoreGeobuk User Service

이 저장소는 Go로 작성된 간단한 RESTful 사용자 서비스입니다. 회원가입과 로그인, 프로필 관리 그리고 목표/활동 관리 API를 제공합니다.

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
- `IMGBB_KEY` (활동 이미지 업로드용)
- `IMGBB_EXPIRATION` (선택) 이미지 만료 시간

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

성공 응답(`201 Created`):
```json
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

성공 응답(`200 OK`):
```json
{
  "message": "로그인 성공 [test@example.com]",
  "result": {
    "name": "테스트",
    "token": "<JWT>",
    "uuid": "f6455c7b-e223-461f-901b-0d2a6932f81b"
  }
}
    
``` 

### 프로필

`POST /api/profile` – 프로필 저장 또는 수정

`GET /api/profile` – 현재 프로필 조회

두 엔드포인트 모두 `Authorization: Bearer <token>` 헤더가 필요합니다.

#### POST 요청 본문(JSON)
`POST /api/profile` 요청 예시:
```json
{
  "profile_image": "https://.../image.png"
}
```

성공 응답(`200 OK`):
```json
{
  "message": "프로필 저장 완료",
  "result": null
}
```

### 목표 (Goals)

모든 목표 관련 엔드포인트는 인증이 필요합니다.

#### 목표 목록 조회
`GET /api/goals`

성공 응답(`200 OK`):
```json
{
  "message": "goals",
  "result": [
    {
      "goal_id": 1,
      "name": "하루 한번 코테풀기",
      "description": "백준 코테문제 1~2개 풀기",
      "tags": ["프로그래밍", "코딩테스트"],
      "weekdays": [2,5]
    }
  ]
}
```

#### 목표 생성
`POST /api/goals`
```json
{
  "name": "string",
  "description": "string",
  "tags": ["tag1", "tag2"],
  "weekdays": [1,3,5]
}
```
성공 응답(`201 Created`):
```json
{
  "message": "goal created",
  "result": {"goal_id": 1}
}
```

#### 목표 수정
`PUT /api/goals/{id}` – 본문의 형식은 목표 생성과 동일

#### 목표 삭제
`DELETE /api/goals/{id}`

#### 활동 완료
`POST /api/goals/{id}/activities`

multipart/form-data 형식으로 `image` 파일과 선택적 `note`, `date`(기본값 오늘)를 전송합니다. 활동은 해당 요일에만 완료할 수 있습니다.

성공 응답(`200 OK`):
```json
{
  "message": "activity completed",
  "result": null
}
```

## 테이블

애플리케이션 시작 시 테이블이 없으면 자동으로 생성됩니다.
- `users` – id, name, email, password hash 저장
- `profiles` – user id와 프로필 이미지 URL 저장
- `goals` – 목표 정보
- `tags` – 태그 목록
- `goal_tags` – 목표와 태그 매핑(N:M), 목표당 최대 5개
- `goal_days` – 목표의 반복 요일
- `activities` – 활동 기록 (이미지/메모)
