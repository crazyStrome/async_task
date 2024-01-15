package async_task

import "time"

type Option func(*options)

func WithConcurrentNum(num int) Option {
	return func(o *options) {
		o.concurrentNum = num
	}
}

func WithTaskNum(num int) Option {
	return func(o *options) {
		o.taskNum = num
	}
}

func WithGlobalTaskTimeout(out time.Duration) Option {
	return func(o *options) {
		o.taskTimeout = out
	}
}
