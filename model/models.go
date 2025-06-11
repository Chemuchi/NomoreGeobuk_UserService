package model

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}

type Profile struct {
	UserID       string
	ProfileImage string
}
