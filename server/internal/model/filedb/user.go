package filedb

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Status   int    `json:"status" gorm:"default:1"`
}
