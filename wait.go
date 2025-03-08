package bus

import "time"

func Wait(conn *Engine) {
	for {
		if conn == nil {
			time.Sleep(time.Second)
			continue
		}
		if conn.conn == nil {
			time.Sleep(time.Second)
			continue
		}
		return
	}
}
