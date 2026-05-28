package ctrl

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
	"github.com/gwyy/WebTrailAI/server/internal/service"
)

// SummaryList 返回当前登录用户最近 30 条每日总结，供插件弹窗列表直接渲染。
func (ctrl *Ctrl) SummaryList(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser == nil {
		response.FailWithMessage("用户身份无效", c)
		return
	}

	list, err := ctrl.srv.ListUserRecentSummaries(c.Request.Context(), jwtUser.ID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessageAndData("获取成功", gin.H{
		"list":  buildSummaryListResponse(list),
		"total": len(list),
	}, c)
}

// SummaryDetail 返回当前登录用户指定日期的每日总结详情，支持 YYYYMMDD 与 YYYY-MM-DD 两种日期格式。
func (ctrl *Ctrl) SummaryDetail(c *gin.Context) {
	jwtUser := ctrl.parseUser(c)
	if jwtUser == nil {
		response.FailWithMessage("用户身份无效", c)
		return
	}

	detailReq := request.SummaryDetail{}
	if err := c.ShouldBindQuery(&detailReq); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	detail, err := ctrl.srv.GetUserSummaryDetail(c.Request.Context(), jwtUser.ID, detailReq.Date)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSummaryDate) || errors.Is(err, service.ErrSummaryNotFound) {
			response.FailWithMessage(err.Error(), c)
			return
		}
		response.FailWithMessage("获取每日总结详情失败", c)
		return
	}

	response.OkWithMessageAndData("获取成功", gin.H{
		"date":      detail.Date,
		"summary":   detail.Summary,
		"parts":     detail.Parts,
		"partCount": len(detail.Parts),
	}, c)
}

func buildSummaryListResponse(list []service.SummaryListItem) []gin.H {
	result := make([]gin.H, 0, len(list))
	for _, item := range list {
		result = append(result, gin.H{
			"date":      item.Date,
			"summary":   item.Summary,
			"partCount": item.PartCount,
		})
	}
	return result
}
