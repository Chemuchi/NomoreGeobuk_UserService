package model

type User struct {
	ID           string
	Email        string
	PasswordHash string
}

type Profile struct {
	UserID       string
	ProfileImage string
}
