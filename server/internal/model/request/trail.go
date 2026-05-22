package request

type TrailAdd struct {
	URL   string `json:"url" form:"url" query:"url" binding:"required"`
	Title string `json:"title" form:"title" query:"title" binding:"required"`
}
