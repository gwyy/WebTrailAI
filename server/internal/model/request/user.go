package request

type UserLogin struct {
	Username string `json:"username" form:"username" query:"username" binding:"required"`
	Password string `json:"password" form:"password" query:"password" binding:"required"`
}

type JwtUser struct {
	ID       int    `json:"id" mapstructure:"id"`
	Username string `json:"username" mapstructure:"username"`
}

type UserRegister struct {
	Username string `json:"username" form:"username" query:"username" binding:"required"`
	Password string `json:"password" form:"password" query:"password" binding:"required"`
}
