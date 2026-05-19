package ctrl

import (
	"errors"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
	"github.com/gwyy/WebTrailAI/server/internal/service"
)

/*
**
注册逻辑
*/

// Register 负责处理注册 HTTP 请求：绑定参数，调用 service 完成注册，并返回脱敏后的用户信息。
func (ctrl *Ctrl) Register(c *gin.Context) {
	registerReq := request.UserRegister{}
	if err := c.ShouldBind(&registerReq); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	user, err := ctrl.srv.Register(c.Request.Context(), &registerReq)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessageAndData("注册成功", gin.H{
		"id":       user.ID,
		"username": user.Username,
	}, c)
}

// 用户登录逻辑
// Authenticator 是 gin-jwt 的登录回调：校验用户名密码，并把成功登录的用户转换成 JWT 需要的身份结构。
func (ctrl *Ctrl) Authenticator(c *gin.Context) (any, error) {
	loginReq := request.UserLogin{}
	if err := c.ShouldBind(&loginReq); err != nil {
		return "", jwt.ErrMissingLoginValues
	}

	user, err := ctrl.srv.Authenticate(c.Request.Context(), &loginReq)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return "", jwt.ErrFailedAuthentication
		}
		ctrl.log.Errorw("登录校验失败", "username", loginReq.Username, "error", err)
		return "", jwt.ErrFailedAuthentication
	}

	//包装返回
	jwtUser := &request.JwtUser{
		ID:       user.ID,
		Username: user.Username,
	}

	return jwtUser, nil
}
