package async_task

import "time"

type TaskOption func(*TaskOptions)

func WithSendTimeout(out time.Duration) TaskOption {
    return func(to *TaskOptions) {
        to.Sendout = out
    }
}

func WithTaskTimeout(out time.Duration) TaskOption {
    return func(to *TaskOptions) {
        to.Timeout = out
    }
}
