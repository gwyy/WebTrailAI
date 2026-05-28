package request

type SummaryDetail struct {
	Date string `form:"date" query:"date" binding:"required"`
}
