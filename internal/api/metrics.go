// 管理Prometheus 的指标

package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

// 定义指标
// 秒杀相关: QPS、成功/失败次数、库存剩余、中奖人数、限流拦截次数

var (
	// 秒杀请求总数
	lotteryRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "interaction_service_lottery_requests_total",
			Help: "Total number of seckill lottery requests",
		},
		[]string{"result"}, // success 或 failed
	)
	// 秒杀库存剩余 (实时库存)
	lotteryInventoryRemaining = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "interaction_service_lottery_inventory_remaining",
			Help: "Remaining inventory for seckill lottery",
		},
	)
	// 秒杀中奖用户总数
	lotteryWinnersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "interaction_service_lottery_winners_total",
			Help: "Total number of seckill winners",
		},
	)
	// 被限流拦截请求总数
	rateLimitBlockedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "interaction_service_rate_limit_blocked_total",
			Help: "Total number of requests blocked by rate limiting",
		},
	)
)

func InitMetrics() {

}

// RecordLotterySuccess 记录秒杀成功
func RecordLotterySuccess() {
	lotteryRequestsTotal.WithLabelValues("success").Inc()
}

// RecordLotteryFailed 记录秒杀失败
func RecordLotteryFailed() {
	lotteryRequestsTotal.WithLabelValues("failed").Inc()
}

// UpdateInventory 更新库存
func UpdateInventory(remaining int64) {
	lotteryInventoryRemaining.Set(float64(remaining))
}

// UpdateWinnersCount 更新中奖人数
func UpdateWinnersCount(count int64) {
	lotteryWinnersTotal.Set(float64(count))
}

// RecordRateLimitBlocked 记录限流拦截
func RecordRateLimitBlocked() {
	rateLimitBlockedTotal.Inc()
}
