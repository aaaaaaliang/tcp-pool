package tcpPool

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPPoolConn struct {
	mu        sync.Mutex
	addr      string
	pool      chan *PoolConnWrapper
	maxConns  int64
	currConns int64 // 当前总连接数
}

func NewTCPConnPool(addr string, maxConns int64) *TCPPoolConn {
	return &TCPPoolConn{
		addr:     addr,
		pool:     make(chan *PoolConnWrapper, maxConns),
		maxConns: maxConns,
	}
}

func (p *TCPPoolConn) Get() (net.Conn, error) {
	select {
	case wrapper := <-p.pool:
		if isDead(wrapper.conn) || time.Since(wrapper.lastUsed) > time.Minute {
			wrapper.conn.Close()
			atomic.AddInt64(&p.currConns, -1)
			return p.newConn()
		}
		return wrapper.conn, nil
	default:
		return p.newConn()
	}
}

func (p *TCPPoolConn) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("nil connection")
	}
	wrapper := &PoolConnWrapper{
		conn:     conn,
		lastUsed: time.Now(),
	}
	select {
	case p.pool <- wrapper:
		return nil
	default:
		conn.Close()
		atomic.AddInt64(&p.currConns, -1)
		return errors.New("pool full")
	}
}

func (p *TCPPoolConn) newConn() (net.Conn, error) {
	if atomic.LoadInt64(&p.currConns) >= p.maxConns {
		return nil, errors.New("当前连接数超过最大允许连接数")
	}
	conn, err := net.DialTimeout("tcp", p.addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	atomic.AddInt64(&p.currConns, 1)
	return conn, nil
}

func (p *TCPPoolConn) Stats() (int64, int64, int64) {
	return p.maxConns,
		atomic.LoadInt64(&p.currConns),
		int64(len(p.pool))
}

func (p *TCPPoolConn) StartCleaner(ctx context.Context, interval time.Duration, ttl time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.cleanExpired(ttl)
			}
		}
	}()
}

func (p *TCPPoolConn) cleanExpired(ttl time.Duration) {
	for {
		select {
		case wrapper := <-p.pool:
			if time.Since(wrapper.lastUsed) > ttl || isDead(wrapper.conn) {
				wrapper.conn.Close()
				atomic.AddInt64(&p.currConns, -1)
				continue
			}
			select {
			case p.pool <- wrapper:
			default:
				wrapper.conn.Close()
				atomic.AddInt64(&p.currConns, -1)
			}
		default:
			return
		}
	}
}

func isDead(conn net.Conn) bool {
	if conn == nil {
		return true
	}
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	if _, err := conn.Write([]byte{}); err != nil {
		return true
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return false
}
