package filedb

type Trail struct {
	Title   string `json:"title" form:"title" gorm:"column:title"`
	URL     string `json:"url" form:"url" gorm:"column:url"`
	UserID  uint   `json:"user_id" form:"user_id" gorm:"column:user_id"`
	Content string `json:"content" form:"content" gorm:"column:content"`
}
