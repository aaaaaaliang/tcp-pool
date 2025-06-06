# TCP连接池模块（Go 实现）

一个高性能、线程安全、支持连接复用、自动清理、并发控制的 TCP 连接池，支持 `Get`/`Put` 接口与可选后台清理器。

---

## 🚀 特性 Features

- ✅ 最大连接数限制（maxConns）
- ✅ 空闲连接复用（使用 channel 实现）
- ✅ 自动过期清理（支持 TTL）
- ✅ Goroutine 安全，原子变量保证并发准确性
- ✅ 状态监控接口（连接数统计）
- ✅ 可选 mock 连接 & Benchmark 性能测试
- ✅ 支持链路追踪扩展（traceID，可用 `context` 接入）

---

## 📦 项目结构

.
├── conn_wrapper.go # 包装连接结构（含 lastUsed 时间）
├── interface.go # 通用连接池接口 ConnPool
├── pool.go # TCPPoolConn 实现
├── pool_test.go # 单元测试与 Benchmark 测试
├── main.go # 示例调用入口
└── go.mod

scss
复制
编辑

---

## 🧪 使用示例（main.go）

```go
pool := tcpPool.NewTCPConnPool("localhost:8080", 10)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()
pool.StartCleaner(ctx, 30*time.Second, time.Minute)

conn, err := pool.Get()
if err != nil {
	log.Fatalf("获取连接失败: %v", err)
}
pool.Put(conn)

max, cur, idle := pool.Stats()
fmt.Printf("最大连接数=%d，当前连接=%d，空闲连接=%d\n", max, cur, idle)
```