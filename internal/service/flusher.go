package service

import (
	"log"
	"time"

	"github.com/stywzn/Go-Interaction-Service/config"
	"github.com/stywzn/Go-Interaction-Service/internal/model"
	"gorm.io/gorm/clause"
)

// LotteryTask 定义要在 Channel 里流转的"抽奖任务包"
type LotteryTask struct {
	UserID int64
	Status int // 1:中奖 0:未中奖
}

// LotteryQueue ：抽奖异步落盘队列
var LotteryQueue = make(chan LotteryTask, 10000)

// StartAsyncFlusher 是点赞系统在后台跳动的“心脏”
// 它会在 main 函数启动时，作为一个独立的 Goroutine 一直运行
func StartAsyncFlusher() {
	// 定义一个定时器，每 5 秒钟滴答一次 (削峰填谷的核心)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 准备一个切片，用来在内存里暂存这 5 秒内收到的任务
	var batchRecords []model.LotteryRecord

	log.Println("后台异步落盘引擎已启动，正在监听 LotteryQueue...")

	for {
		// select 多路复用，同时监听任务管道和定时器
		select {
		case task := <-LotteryQueue:
			// 从管道里拿到了一个抽奖任务，转换成 GORM 模型放到推车里
			record := model.LotteryRecord{
				UserID: task.UserID,
				Status: task.Status,
			}
			batchRecords = append(batchRecords, record)

			if len(batchRecords) >= 500 {
				flushLotteryToDB(batchRecords)
				batchRecords = nil // 清空推车，准备接下一批
			}

		case <-ticker.C:
			// 5 秒时间到了！不管推车里装了多少条（哪怕只有 1 条），也必须发车写进数据库
			if len(batchRecords) > 0 {
				flushLotteryToDB(batchRecords)
				batchRecords = nil // 清空推车
			}
		}
	}
}

// flushLotteryToDB 执行真正的 MySQL 批量写入操作（抽奖数据）
func flushLotteryToDB(records []model.LotteryRecord) {
	if len(records) == 0 {
		return
	}

	// 抽奖记录使用幂等性唯一索引 uk_user_lottery，防止同一用户多次中奖
	err := config.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).CreateInBatches(records, 100).Error

	if err != nil {
		log.Printf("[Flusher] 批量写入抽奖数据到 MySQL 失败: %v\n", err)
	} else {
		log.Printf("[Flusher] 成功将 %d 条抽奖数据异步落盘到 MySQL!\n", len(records))
	}
}
