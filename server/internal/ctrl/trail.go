package ctrl

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
)

func (ctrl *Ctrl) TrailList(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser != nil {
		msg := fmt.Sprintf("id:%d 用户名：%s list", jwtUser.ID, jwtUser.Username)
		response.OkWithMessage(msg, c)
	}
}
