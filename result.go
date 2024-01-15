package async_task

import (
	"errors"
	"time"
)

type Result struct {
	request    interface{}
	response   interface{}
	taskStatus int32
	taskID     string
	start      time.Time
	end        time.Time
	errorMsg   string
	isError    bool
}

func (r *Result) GetResponse() interface{} {
	return r.response
}

func (r *Result) GetCost() time.Duration {
    if !r.Done() {
        return -1
    }
	return r.end.Sub(r.start)
}

func (r *Result) Done() bool {
	return r.taskStatus == TaskStatusDone
}

func (r *Result) GetError() error {
    if r.isError {
        return errors.New(r.errorMsg)
    }
    return nil
}
