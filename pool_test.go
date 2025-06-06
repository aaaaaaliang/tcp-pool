package tcpPool

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type mockConn struct {
	closed bool
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return 0, nil }
func (m *mockConn) Close() error                       { m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewTCPConnPool(t *testing.T) {
	pool := NewTCPConnPool("localhost:9999", 2)

	conn1 := &mockConn{}
	atomic.AddInt64(&pool.currConns, 1)
	_ = pool.Put(conn1)

	conn2 := &mockConn{}
	atomic.AddInt64(&pool.currConns, 1)
	_ = pool.Put(conn2)

	// 超过最大连接数测试
	conn3 := &mockConn{}
	err := pool.Put(conn3)
	if err == nil {
		t.Errorf("expected error when pool is full")
	}

	max, cur, idle := pool.Stats()
	if cur != 2 || idle != 2 {
		t.Errorf("unexpected stats: max=%d, cur=%d, idle=%d", max, cur, idle)
	}
}

func TestPoolCleaner(t *testing.T) {
	pool := NewTCPConnPool("localhost:9999", 5)

	// 放入过期连接
	conn := &mockConn{}
	atomic.AddInt64(&pool.currConns, 1)
	_ = pool.Put(conn)

	// 启动清理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.StartCleaner(ctx, 200*time.Millisecond, 100*time.Millisecond)

	time.Sleep(1 * time.Second)

	_, cur, idle := pool.Stats()
	if cur != 0 || idle != 0 {
		t.Errorf("连接未被清理: 当前连接数=%d, 空闲数=%d", cur, idle)
	}
}

func BenchmarkGetPut(b *testing.B) {
	pool := NewTCPConnPool("localhost:9999", 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.StartCleaner(ctx, 10*time.Second, 30*time.Second)

	// 预填充部分连接
	for i := 0; i < 50; i++ {
		conn := &mockConn{}
		atomic.AddInt64(&pool.currConns, 1)
		_ = pool.Put(conn)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.Get()
			if err != nil {
				b.Errorf("Get failed: %v", err)
				continue
			}
			// 模拟轻量操作
			time.Sleep(100 * time.Microsecond)
			_ = pool.Put(conn)
		}
	})
}
