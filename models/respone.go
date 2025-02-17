package models

type ForumResponse struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Photo        string `json:"photo"`
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Profile      string `json:"profile"`
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	RelativeTime string `json:"relative_time"`
}

type ForumCreateResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Photo        string    `json:"photo"`
	Username     string    `json:"username"`
	CategoryName string    `json:"category_name"`
	Tags         []Tag     `json:"tags"`
	CreatedAt    string    `json:"created_at"`
	RelativeTime string    `json:"relative_time"`
}

