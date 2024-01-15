package async_task

import (
	"context"
)

// Executor async_task 暴露出去的接口
type Executor interface {
	// SendTask 用户发送任务
	// ctx 只用于控制发送任务的耗时
	// taskID 为标识该任务的唯一 ID
	// req 为该任务执行的请求参数
	// options 为该任务执行时的相关参数，会覆盖 New 时的全局参数
	SendTask(ctx context.Context, taskID string, req interface{}, opts ...TaskOption) error
	// GetResult 用户获取任务结果
	// ctx 只用于控制查询任务的耗时，具体耗时依赖用户提供的存储插件
	// taskID async_task 通过该 ID 识别任务执行结果
	// Result 可以获取任务是否完成、任务结果、任务耗时等数据
	GetResult(ctx context.Context, taskID string) (*Result, error)
}

// Storager 是用户提供的存储插件
type Storager interface {
	// Set 用来获取任务的数据，包含请求和响应
	Set(ctx context.Context, taskID string, task *Task) error
	// Get 通过任务 taskID 获取任务的详情，err 是查数据的报错，而不是执行任务报的错
	Get(ctx context.Context, taskID string) (task *Task, err error)
}
