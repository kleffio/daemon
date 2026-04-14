package workers

import (
	"context"
	"net"
	"time"
)

// waitForPort dials addr (host:port) repeatedly until the TCP connection
// succeeds, the timeout elapses, or ctx is cancelled. It returns nil only
// when the port is actually accepting connections.
func waitForPort(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		if time.Now().After(deadline) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}
