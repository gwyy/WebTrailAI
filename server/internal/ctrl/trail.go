package ctrl

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
)

func (ctrl *Ctrl) TrailList(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser != nil {
		msg := fmt.Sprintf("id:%d 用户名：%s list", jwtUser.ID, jwtUser.Username)
		response.OkWithMessage(msg, c)
	}
}

/*
*
添加 url
*/
func (ctrl *Ctrl) TrailAdd(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser == nil {
		response.FailWithMessage("用户身份无效", c)
		return
	}

	trailReq := request.TrailAdd{}
	if err := c.ShouldBind(&trailReq); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := ctrl.srv.AddTrail(c.Request.Context(), jwtUser.ID, &trailReq)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if result.Filtered {
		response.OkWithMessageAndData("浏览记录已过滤", gin.H{
			"trail":    nil,
			"total":    result.Total,
			"filtered": true,
			"reason":   result.Reason,
		}, c)
		return
	}

	response.OkWithMessageAndData("添加成功", gin.H{
		"trail":    buildTrailResponse(result.Trail),
		"total":    result.Total,
		"filtered": false,
	}, c)
}

func buildTrailResponse(trail *filedb.Trail) gin.H {
	if trail == nil {
		return nil
	}

	return gin.H{
		"title":      trail.Title,
		"url":        trail.URL,
		"user_id":    trail.UserID,
		"created_at": trail.CreatedAt,
	}
}

/*
*
删除今天全部的 url
*/
func (ctrl *Ctrl) CleanTodayTrail(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser == nil {
		response.FailWithMessage("用户身份无效", c)
		return
	}

	deleted, err := ctrl.srv.CleanTodayTrail(c.Request.Context(), jwtUser.ID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessageAndData("清空成功", gin.H{
		"deleted": deleted,
		"total":   0,
	}, c)
}
