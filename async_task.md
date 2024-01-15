# 同步任务转异步框架

## 1. 背景
最近在把公众号接入 OpenAI，主要链路如下：
1. 用户在公众号发消息
2. 公众号将消息转发给服务器
3. 服务器调用 OpenAI 获取文本消息的响应
4. 服务器返回消息给公众号
5. 公众号回复消息给用户

公众号在转发消息给服务器时，会设置 5s 的超时，如果 5s 内服务器没有回复，公众号会断开链接并重新发起请求，总共重试 3 次。
接入的 OpenAI 的响应时间有时会超过 10s，常规的消息转发模式会导致公众号重试请求三次且都无法获取到响应消息。此外，OpenAI 也有频控，如果单纯使用失败重试的逻辑，OpenAI 的有效使用次数会降低为原来的四分之一。

![](https://github.com/crazyStrome/myPhote/blob/master/async_task_wechat1.svg)

为了充分利用 OpenAI 的可用次数，考虑到公众号重试时会携带相同的 MsgID，笔者计划把该同步链路改造成异步链路，这样即使第一次消息处理超时，后续的重试也不会消耗 OpenAI 的可用次数，并且可以在后续重试时返回响应的消息。

具体的调用链路如下所示。用户的一次消息处理链路中，共调用一次 OpenAI 获取响应。

![](https://github.com/crazyStrome/myPhote/blob/master/async_task_wechat2.svg)

笔者的服务器是单机服务器，很容易就实现了异步链路的改造。主要思路是本地缓存+异步处理 goroutine。

后来笔者想到是否可以提取出来一个同步转异步的轻量框架，开发者只需要注册实现其同步逻辑和存储逻辑，就可以异步获取任务结果。这就是本文的目标，本文中会介绍该框架的结构和运行逻辑，然后分别介绍下单机版和集群版接入方式。（单机版即上文所说的公众号链路）。

## 2. 思路
该框架和用户代码交互的具体逻辑如下，框架名为 async_task。

![](https://github.com/crazyStrome/myPhote/blob/master/async_task1.svg)

用户代码和 async_task 交互流程如下：
1. 用户将需要执行的任务发送给 async_task 后并直接返回
2. async_task 收到任务后，新起一个流程执行该任务，并当任务执行成功后存储其结果
3. 用户发送完任务后可以执行其他业务逻辑
4. 用户可通过 async_task 的接口查询任务是否完成，并获取执行结果

async_task 的定位为轻量级的异步框架，因此其执行任务的逻辑和储存任务数据的逻辑都需要由用户注册实现：
* 首先，用户代码需要实现任务同步执行的逻辑，并注册到 async_task 框架中。
* 其次，async_task 作为轻量级框架，不直接与存储进行交互，因此需要用户将与存储交互的行为注册为存储层插件。用户可以根据业务特性，选择不同的存储接入该框架。

同步执行任务逻辑通过函数形式注册进 async_task 中：

```go
// HandleFunc 即用户定义的处理实际业务的同步函数
// ctx 可用于超时控制，从 New 链路一直透传到这里，超时可以通过 New 的 Option 设置
// 为了提高效率，不做序列化和反序列化处理，用户直接通过类型断言获取请求体
// req 即用户 SendTask 时传入的请求体
// rsp 即用户 GetResult 时获取到的结果
type HandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)
```

存储插件也通过接口形式注册进 async_task，其中 Task 标识一个任务的数据，包括请求、响应以及一些业务辅助信息。

```go
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

// Storager 是用户提供的存储插件
type Storager interface {
	// Set 用来获取任务的数据，包含请求和响应
	Set(ctx context.Context, taskID string, task *Task) error
	// Get 通过任务 taskID 获取任务的详情，err 是查数据的报错，而不是执行任务报的错
	Get(ctx context.Context, taskID string) (task *Task, err error)
}
```

async_task 也会提供两个函数，一个用于发送任务，一个用于查询任务；两个函数都需要通过一个唯一键来关联任务。

这两个函数也以接口的形式暴露给用户。

```go
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
```

GetResult 接口返回的 Result 中封装了 async_task 的执行信息，可以判断任务是否执行完成，如果没有该任务的话，会直接报 error 出来，而不需要通过 Result 来做判断。

## 3. 实现
接下来我们依照用户代码和 async_task 交互的顺序来实现 async_task 框架。

### 3.1. 初始化
如果需要使用 async_task，必须初始化一个实例。这样，不同的实例可以用来执行不同的异步逻辑。

初始化时，用户代码传递一些可选参数和必选参数。
* 必选参数包括：执行任务的函数，存储插件
* 可选参数包括：执行任务的超时，任务执行的并发控制，任务队列长度限制等，这些通过 Option 模式设置

函数签名如下：
```go 
func New(
	ctx context.Context,
	handle HandleFunc,
	storage Storager,
	os ...Option) Executor 
```

传入的 ctx 用来控制整个链路的生命周期；handle 则是实际执行任务的逻辑；storage 是储存任务数据的插件；os 则是一些辅助的可选参数。

如果用户不设置的话，默认任务队列长度 100，并发控制 20。

初始化时，会异步启动一个消费者，它会消费一个 channel 中的 Task 数据，并异步执行该 Task。

### 3.2. 发送任务
用户代码通过 async_task 的 SendTask 发送任务，该框架为了在轻量级的基础上保证可扩展性，也是通过 TaskOption 的形式扩展一些辅助的可选参数，例如任务执行超时控制。

函数定义如下：
```go
func (as *asyncImpl) SendTask(ctx context.Context,
	taskID string, req interface{}, tos ...TaskOption) error 
```

这个函数中的逻辑很简单，就是将用户的请求参数封装为一个 Task，并写入 channel 中。channel 的另一端就是上节说的消费者。

任务队列也是有长度限制的，如果用户代码发送任务的速度超过了 async_task 执行任务的速度到一定程度，任务队列会写满。

那么此时，用户代码不可能无限期等在 SendTask 这里，所以我们提供一个发送超时，如果用户不设置该超时时，async_task 默认使用 100ms 作为发送超时。

```go
    tick := time.After(opts.Sendout)
	select {
	case <-tick:
		return errors.New("too many tasks to process")
	case as.taskCh <- task:
        as.storager.Set(ctx, taskID, task)
		return nil
	}
```

发送完任务后，async_task 会将该任务通过存储插件持久化。

### 3.3. 消费者处理任务
初始化 async_task 时，会启动一个消费者，不停地接收 channel 中的 Task。

每收到一个 Task，消费者都会新起一个 goroutine 来处理该任务。New 函数传入的 ctx 则用于消费者的优雅退出。
```go
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
```

handleTask 执行时，则会设置任务执行超时、记录执行耗时、储存任务结果。
```go
	go func() {
		defer func() {
			<-as.concurrentCh
            close(done)
		}()
		task.StartTime = time.Now()
		rsp, err := as.handle(ctx, task.Request)
		task.DoneTime = time.Now()
		if err != nil {
			task.Error = err.Error()
		}
		task.Response = rsp
        task.Status = TaskStatusDone
        as.storager.Set(ctx, task.TaskID, task)
	}()
```
## 4. 应用
框架的使用也是很简单的，只需要用户代码里定义好执行函数和存储插件，剩下的就是调接口了。
### 4.1. 单机版
单机版的执行函数就很简单，直接 sleep 10s 模拟长耗时任务。
```go
func localHandle(ctx context.Context, req interface{}) (interface{}, error) {
	time.Sleep(10 * time.Second)
	fmt.Printf("localHandle:%v", toJSON(req))
	return map[string]interface{}{"result": "OK"}, nil
}
```

存储插件就直接用一个 map 来实现。但是单纯用 map 在单机上还会存在内存泄漏问题，因此建议使用 gocache 来实现。
```go
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
```

用户使用时先初始化，然后发送任务，等待 5s 后获取一次结果，然后再等待 6s 后获取一次结果，此时会获取到执行结果。
```go
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
```
### 4.2. 集群版
集群版留待实现。不过大致思路还是有的，执行函数根据业务逻辑实现，存储插件可以用 redis 或 mysql 实现，具体选用逻辑可参考业务特性。

感兴趣的读者可以实现一下，将代码放在 example 中。

## 最后
async_task 只是出于兴趣写出来的一个小框架，感兴趣的读者可以提出改正意见或者提 MR。
