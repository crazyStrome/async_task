package async_task

import "context"

// HandleFunc 即用户定义的处理实际业务的同步函数
// ctx 可用于超时控制，从 New 链路一直透传到这里，超时可以通过 New 的 Option 设置
// 为了提高效率，不做序列化和反序列化处理，用户直接通过类型断言获取请求体
// req 即用户 SendTask 时传入的请求体
// rsp 即用户 GetResult 时获取到的结果
type HandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)
