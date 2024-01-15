package async_task

import "time"

type TaskOptions struct {
	Timeout time.Duration // 任务执行超时
	Sendout time.Duration // 发送任务超时
}
