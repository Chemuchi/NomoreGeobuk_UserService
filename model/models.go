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

type Goal struct {
	ID          int64    `db:"goal_id" json:"goal_id"`
	Name        string   `db:"name" json:"name"`
	Description string   `db:"description" json:"description"`
	Tags        []string `json:"tags"`
	Weekdays    []int    `json:"weekdays"`
}
