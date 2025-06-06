package tcpPool

import (
	"context"
	"net"
	"time"
)

type ConnPool interface {
	Get() (net.Conn, error)
	Put(net.Conn) error
	Stats() (max int64, current int64, idle int64)
	StartCleaner(ctx context.Context, interval, ttl time.Duration)
}
