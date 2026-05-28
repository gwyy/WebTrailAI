package filedb

type Trail struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	InnerText string `json:"inner_text"`
	UserID    int    `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
}
