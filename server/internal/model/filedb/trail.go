package filedb

type Trail struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	UserID    int    `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
}
