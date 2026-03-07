package ctrl

import (
	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
)

// 用户登录逻辑
func (ctrl *Ctrl) Authenticator(c *gin.Context) (any, error) {
	loginReq := request.UserLogin{}
	if err := c.ShouldBind(&loginReq); err != nil {
		return "", jwt.ErrMissingLoginValues
	}

	//判断账号密码是否正确
	if loginReq.Username != "ebwaaa" || loginReq.Password != "aaaaaa" {
		return "", jwt.ErrFailedAuthentication
	}
	user := &filedb.User{
		ID:       2,
		Username: "ebwaaa",
	}
	if user == nil {
		return nil, jwt.ErrFailedAuthentication
	}

	//包装返回
	jwtUser := &request.JwtUser{
		ID:       user.ID,
		Username: user.Username,
	}

	return jwtUser, nil
}
