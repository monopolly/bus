package bus

import "time"

// WaitTimeout bounds how long Wait blocks while the connection is reconnecting
var WaitTimeout = 30 * time.Second

// Wait blocks until the engine is connected to the server.
// It returns immediately for a nil engine or an engine without a
// connection (those can never become ready), and gives up after WaitTimeout
// during a long reconnect so the caller gets an error from the next call
// instead of hanging forever.
func Wait(conn *Engine) {
	if conn == nil || conn.conn == nil {
		return
	}
	deadline := time.Now().Add(WaitTimeout)
	for !conn.conn.IsConnected() && !conn.conn.IsClosed() {
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
