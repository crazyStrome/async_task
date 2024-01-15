package async_task

import "time"

const (
	TaskStatusInit    = 0
	TaskStatusProcess = 1
	TaskStatusDone    = 2
)

// Task 包含待执行的任务以及结果
type Task struct {
	TaskID      string      `json:"task_id"`
	Status      int32       `json:"status"`
	Request     interface{} `json:"request"`
	Response    interface{} `json:"reponse"`
	CreateTime  time.Time   `json:"create_time"` // 任务创建时间
	StartTime   time.Time   `json:"start_time"`  // 任务开始时间
	DoneTime    time.Time   `json:"done_time"`   // 任务结束时间
	Error       string      `json:"error"`       
    IsError bool `json:"is_error"` // 是否执行时报错，如果执行时报错，错误信息就是 error
	TaskOptions TaskOptions `json:"task_options"`
}
