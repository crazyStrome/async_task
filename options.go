package async_task

import "time"

type options struct {
	concurrentNum int // 并发执行任务的量
	taskNum       int // 待处理的任务数量上限
	taskTimeout   time.Duration
}
