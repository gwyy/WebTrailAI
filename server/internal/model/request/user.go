package request

type UserLogin struct {
	Username string `json:"username" form:"username" query:"username" binding:"required"`
	Password string `json:"password" form:"password" query:"password" binding:"required"`
	Email    string `json:"email" form:"email" query:"email"`
	ID       int    `json:"id" form:"id" query:"id"`
}

type JwtUser struct {
	ID       int    `json:"id" mapstructure:"id"`
	Username string `json:"username" mapstructure:"username"`
}

type UserRegister struct {
	Username   string `json:"username" form:"username" query:"username" binding:"required"`
	Password   string `json:"password" form:"password" query:"password" binding:"required"`
	Email      string `json:"email" form:"email" query:"email" binding:"required"`
	VerifyCode string `json:"verify_code" form:"verify_code" query:"verify_code" binding:"required"`
}

type UserChangePassword struct {
	OldPassword string `json:"old_password" form:"old_password" query:"old_password" binding:"required"`
	NewPassword string `json:"new_password" form:"new_password" query:"new_password" binding:"required"`
	VerifyCode  string `json:"verify_code" form:"verify_code" query:"verify_code" binding:"required"`
}

type UserChangeEmail struct {
	Email      string `json:"email" form:"email" query:"email" binding:"required"`
	VerifyCode string `json:"verify_code" form:"verify_code" query:"verify_code" binding:"required"`
}

type UserForgetPassword struct {
	Email      string `json:"email" form:"email" query:"email" binding:"required"`
	Password   string `json:"password" form:"password" query:"password" binding:"required"`
	VerifyCode string `json:"verify_code" form:"verify_code" query:"verify_code" binding:"required"`
}

type VerifyCode struct {
	Email string `json:"email" form:"email" query:"email" binding:"required"`
}
