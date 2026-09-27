package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Interaction-Service/internal/service"
)

// HandleJoinLottery 参与抽奖
func HandleJoinLottery(c *gin.Context) {
	// 从 JWT 中获取用户 ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "请先登录"})
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "用户信息解析失败"})
		return
	}

	// 调用抽奖服务
	won, err := service.JoinLottery(c.Request.Context(), userID)
	if err != nil {
		RecordLotteryFailed()
		c.JSON(http.StatusConflict, gin.H{
			"code": 409,
			"msg":  err.Error(),
		})
		return
	}

	if won {
		RecordLotterySuccess()
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "恭喜中奖！获得 4GB 存储配额",
			"data": gin.H{"reward_gb": 4},
		})
	}
}

// HandleLotteryStatus 查看抽奖状态
func HandleLotteryStatus(c *gin.Context) {
	status, err := service.GetLotteryStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}

	// 更新 Prometheus 指标
	if remaining, ok := status["remaining_slots"].(int64); ok {
		UpdateInventory(remaining)
	}
	if winners, ok := status["winners_count"].(int64); ok {
		UpdateWinnersCount(winners)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": status,
	})
}
