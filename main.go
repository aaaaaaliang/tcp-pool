package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

type TCPPoolConn struct {
	mu       sync.Mutex
	addr     string
	pool     chan net.Conn
	maxConns int
}

func NewTCPConnPool(addr string, maxConns int) *TCPPoolConn {
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
		return errors.New("pool full")
	}
}

func (p *TCPPoolConn) newConn() (net.Conn, error) {
	return net.DialTimeout("tcp", p.addr, 2*time.Second)
}

func isDead(conn net.Conn) bool {
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	var buf [1]byte
	_, err := conn.Read(buf[:])
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			_ = conn.SetReadDeadline(time.Time{})
			return false
		}
		return true
	}
	return false
}
