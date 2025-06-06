package tcpPool

import (
	"net"
	"time"
)

type PoolConnWrapper struct {
	conn     net.Conn
	lastUsed time.Time
}
