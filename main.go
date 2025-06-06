package main

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPPoolConn struct {
	mu        sync.Mutex
	addr      string
	pool      chan net.Conn
	maxConns  int64
	currConns int64 // 当前总连接数
}

func NewTCPConnPool(addr string, maxConns int64) *TCPPoolConn {
	return &TCPPoolConn{
		addr:     addr,
		pool:     make(chan net.Conn, maxConns),
		maxConns: maxConns,
	}
}

func (p *TCPPoolConn) Get() (net.Conn, error) {
	select {
	case conn := <-p.pool:
		if isDead(conn) {
			conn.Close()
			return p.newConn()
		}
		return conn, nil
	default:
		return p.newConn()
	}
}

func (p *TCPPoolConn) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("nil connection")
	}
	select {
	case p.pool <- conn:
		return nil
	default:
		conn.Close()
		atomic.AddInt64(&p.currConns, -1)
		return errors.New("pool full")
	}
}

func (p *TCPPoolConn) newConn() (net.Conn, error) {
	if atomic.LoadInt64(&p.currConns) > p.maxConns {
		return nil, errors.New("当前连接数超过最大允许连接数")
	}

	conn, err := net.DialTimeout("tcp", p.addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	atomic.AddInt64(&p.currConns, 1)
	return conn, nil
}

func isDead(conn net.Conn) bool {
	if conn == nil {
		return true
	}
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	//var buf [1]byte
	//_, err := conn.Read(buf[:])
	//if err != nil {
	//	if e, ok := err.(net.Error); ok && e.Timeout() {
	//		_ = conn.SetReadDeadline(time.Time{})
	//		return false
	//	}
	//	return true
	//}
	if _, err := conn.Write([]byte{}); err != nil {
		return true
	}
	return false
}
