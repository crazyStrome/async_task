package example

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/crazyStrome/async_task"
)

type localStorageImpl struct {
	data map[string]*async_task.Task
}

func (s *localStorageImpl) Set(ctx context.Context, taskID string, task *async_task.Task) error {
	if s.data == nil {
        s.data = map[string]*async_task.Task{}
	}
	s.data[taskID] = task
    fmt.Printf("local storage store:%v task:%v\n", taskID, toJSON(task))
	return nil
}

func (s *localStorageImpl) Get(ctx context.Context, taskID string) (*async_task.Task, error) {
	return s.data[taskID], nil
}

func localHandle(ctx context.Context, req interface{}) (interface{}, error) {
	time.Sleep(10 * time.Second)
	fmt.Printf("localHandle:%v", toJSON(req))
	return map[string]interface{}{"result": "OK"}, nil
}

func toJSON(val interface{}) string {
	bs, _ := json.Marshal(val)
	return string(bs)
}

func TestLocalAsyncTask(t *testing.T) {
	ctx := context.Background()
	async := async_task.New(ctx, localHandle, &localStorageImpl{})

	taskID := "taskID_test_loacal"
	async.SendTask(ctx, taskID, map[string]interface{}{"req": "test"})
	fmt.Println("SendTask done")
	time.Sleep(5 * time.Second)

	result, err := async.GetResult(ctx, taskID)
    fmt.Printf("after 5 second, result:%v, err:%v, cost:%v, done:%v, process err:%v", 
    toJSON(result.GetResponse()), err, result.GetCost(), result.Done(), result.GetError())

    time.Sleep(6 * time.Second)

	result, err = async.GetResult(ctx, taskID)
    fmt.Printf("after 11 second, result:%v, err:%v, cost:%v, done:%v, process err:%v", toJSON(result.GetResponse()), err, result.GetCost(), result.Done(), result.GetError())	
}
