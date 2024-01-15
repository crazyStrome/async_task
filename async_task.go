package async_task

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

func New(
	ctx context.Context,
	handle HandleFunc,
	storage Storager,
	os ...Option) Executor {
	opts := &options{}
	for _, o := range os {
		o(opts)
	}
	if opts.concurrentNum == 0 {
		opts.concurrentNum = 20
	}
	if opts.taskNum == 0 {
		opts.taskNum = 100
	}
	asp := &asyncImpl{
		handle:   handle,
		storager: storage,
		opts:     opts,
	}
	asp.taskCh = make(chan *Task, opts.taskNum)
	asp.concurrentCh = make(chan struct{}, opts.concurrentNum)

	go asp.process(ctx)
	return asp
}

type asyncImpl struct {
	handle   HandleFunc
	storager Storager

	taskCh       chan *Task
	concurrentCh chan struct{}
	opts         *options

	done chan struct{}
}

func (as *asyncImpl) SendTask(ctx context.Context,
	taskID string, req interface{}, tos ...TaskOption) error {
	opts := &TaskOptions{}
	for _, o := range tos {
		o(opts)
	}
	if opts.Sendout == 0 {
		// 默认发送 100 ms 超时
		opts.Sendout = 100 * time.Millisecond
	}
	task := &Task{
		TaskID:      taskID,
		Request:     req,
		Status:      TaskStatusInit,
		CreateTime:  time.Now(),
		TaskOptions: *opts,
	}
	tick := time.After(opts.Sendout)
	select {
	case <-tick:
		return errors.New("too many tasks to process")
	case as.taskCh <- task:
		as.storager.Set(ctx, taskID, task)
		return nil
	}
}

func (as *asyncImpl) GetResult(ctx context.Context, taskID string) (*Result, error) {
	task, err := as.storager.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task:%v not found", taskID)
	}
	return &Result{
		request:    task.Request,
		response:   task.Response,
		taskStatus: task.Status,
		taskID:     task.TaskID,
		start:      task.StartTime,
		end:        task.DoneTime,
		errorMsg:   task.Error,
		isError:    task.IsError,
	}, nil
}

func (as *asyncImpl) process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-as.taskCh:
			as.handleTask(ctx, task)
		}
	}
}

func (as *asyncImpl) handleTask(ctx context.Context, task *Task) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("handleTask panic:%+v, stack:%s", rec, debug.Stack())
		}
	}()
	as.concurrentCh <- struct{}{}
	task.Status = TaskStatusProcess
	if err := as.storager.Set(ctx, task.TaskID, task); err != nil {
		fmt.Printf("SetTask err:%v, taskID:%v", err, task.TaskID)
		return
	}

	var timeout time.Duration
	if as.opts.taskTimeout > 0 {
		timeout = as.opts.taskTimeout
	}
	if task.TaskOptions.Timeout > 0 {
		timeout = task.TaskOptions.Timeout
	}
	done := make(chan struct{})
	var nctx context.Context
	if timeout > 0 {
		var cancel context.CancelFunc
		nctx, cancel = context.WithTimeout(ctx, timeout)
		go func() {
			select {
			case <-as.done:
				cancel()
			case <-done:
				return
			}
		}()
	}
	go func() {
		defer func() {
			<-as.concurrentCh
			close(done)
		}()
		task.StartTime = time.Now()
		rsp, err := as.handle(nctx, task.Request)
		task.DoneTime = time.Now()
		if err != nil {
			task.Error = err.Error()
		}
		task.Response = rsp
		task.Status = TaskStatusDone
		if err := as.storager.Set(ctx, task.TaskID, task); err != nil {
			fmt.Printf("SetTask done err:%v, taskID:%v", err, task.TaskID)
		}
	}()
}
