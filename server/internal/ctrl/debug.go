package ctrl

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
)

const summaryDebugAction = "run_yesterday_summary"

// DebugRunSummary 通过受控 GET 参数手动触发一次昨日浏览总结生成，便于本地和临时排障。
// http://localhost:3459/debug/summary?action=run_yesterday_summary&token=123
func (ctrl *Ctrl) DebugRunSummary(c *gin.Context) {
	if !ctrl.cfg.GetBool("summary.debug-enabled") {
		response.FailWithMessage("summary 调试接口未启用", c)
		return
	}

	if strings.TrimSpace(c.Query("action")) != summaryDebugAction {
		response.FailWithMessage("summary 调试动作无效", c)
		return
	}

	expectedToken := strings.TrimSpace(ctrl.cfg.GetString("summary.debug-token"))
	if expectedToken == "" {
		response.FailWithMessage("summary 调试 token 未配置", c)
		return
	}
	if c.Query("token") != expectedToken {
		response.FailWithMessage("summary 调试 token 无效", c)
		return
	}

	// 调试接口会触发较长的大模型任务，不能被浏览器刷新、curl 断开或反代读超时连带取消。
	result, err := ctrl.srv.RunYesterdaySummaryOnce(context.WithoutCancel(c.Request.Context()))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessageAndData("summary 生成已执行", gin.H{
		"date":          result.Date,
		"user_count":    result.UserCount,
		"success_count": result.SuccessCount,
		"skipped_count": result.SkippedCount,
		"failed_count":  result.FailedCount,
	}, c)
}
