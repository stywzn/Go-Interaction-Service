package service

import (
	"context"
	"fmt"
	"log"

	"github.com/stywzn/Go-Interaction-Service/config"
	"github.com/stywzn/Go-Interaction-Service/internal/model"
)

// LotteryConfig 抽奖配置
type LotteryConfig struct {
	TotalSlots int64
	RewardGB   int64
}

var LotteryConf = LotteryConfig{
	TotalSlots: 4,
	RewardGB:   4,
}

// JoinLottery 用户参与抽奖逻辑
// 使用 Redis Lua 脚本保证原子性：
// 1. 检查用户是否已中过一次奖（幂等）
// 2. 如果库存 > 0，原子扣减库存并标记用户中奖
func JoinLottery(ctx context.Context, userID int64) (bool, error) {
	key := "lottery:inventory"
	userSetKey := "lottery:winners"

	// Lua 脚本：原子操作实现"检查+扣减+去重"
	luaScript := `
if redis.call('SISMEMBER', KEYS[2], ARGV[1]) == 1 then
  return 0
end

local stock = redis.call('GET', KEYS[1])
if not stock or tonumber(stock) <= 0 then
  return -1
end

redis.call('DECR', KEYS[1])
redis.call('SADD', KEYS[2], ARGV[1])
return 1
`

	result, err := config.Redis.Eval(ctx, luaScript, []string{key, userSetKey}, userID).Result()
	if err != nil {
		return false, fmt.Errorf("抽奖引擎异常: %v", err)
	}

	switch result {
	case int64(1):
		// 中奖了！异步落库，同时更新用户的存储配额
		go func() {
			updateUserQuota(userID)
			// 改为使用异步队列，而不是直接调用 recordLottery
			select {
			case LotteryQueue <- LotteryTask{UserID: userID, Status: 1}:
				log.Printf("[入队成功] 用户 %d 的抽奖结果入队\n", userID)
			default:
				log.Println("[严重告警] 抽奖队列已满！触发降级，记录丢弃！")
			}
		}()
		return true, nil

	case int64(0):
		return false, fmt.Errorf("您已经中过奖了，每位用户仅限中奖一次")

	case int64(-1):
		return false, fmt.Errorf("库存已空，下次再来吧！")

	default:
		return false, fmt.Errorf("未知错误")
	}
}

// InitLotteryInventory 初始化抽奖库存
func InitLotteryInventory(ctx context.Context, slots int64) error {
	key := "lottery:inventory"
	return config.Redis.Set(ctx, key, slots, 0).Err()
}

// GetLotteryStatus 获取当前抽奖库存和中奖人数
func GetLotteryStatus(ctx context.Context) (map[string]interface{}, error) {
	key := "lottery:inventory"
	userSetKey := "lottery:winners"

	stock, err := config.Redis.Get(ctx, key).Int64()
	if err != nil {
		stock = 0
	}

	winners, err := config.Redis.SCard(ctx, userSetKey).Result()
	if err != nil {
		winners = 0
	}

	return map[string]interface{}{
		"total_slots":     LotteryConf.TotalSlots,
		"remaining_slots": stock,
		"winners_count":   winners,
	}, nil
}

// 辅助函数：异步更新用户配额
func updateUserQuota(userID int64) {
	if err := config.DB.Model(&model.User{}).
		Where("id = ?", userID).
		Update("storage_quota", 5+LotteryConf.RewardGB).Error; err != nil {
		log.Printf("[抽奖] 更新用户 %d 配额失败: %v\n", userID, err)
	} else {
		log.Printf("[抽奖] 用户 %d 增加 %d GB 存储配额\n", userID, LotteryConf.RewardGB)
	}
}

// 辅助函数：记录抽奖结果（已废弃，改为异步队列）
// func recordLottery(userID int64, status int) {
//	record := model.LotteryRecord{
//		UserID: userID,
//		Status: status,
//	}
//	if err := config.DB.Create(&record).Error; err != nil {
//		log.Printf("[抽奖] 记录用户 %d 抽奖结果失败: %v\n", userID, err)
//	}
// }
