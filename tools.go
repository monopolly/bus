package bus

import (
	"encoding/binary"
	"strings"
	"unicode"

	"github.com/nats-io/nats.go/jetstream"
)

// unique builds a durable consumer name from a subject.
// Letters and digits are kept as is (compatible with consumers
// created by earlier versions), wildcards are encoded so that
// "signup.*" and "signup.>" do not collide.
func unique(subj string) (durable string) {
	var res strings.Builder
	for _, x := range subj {
		switch {
		case unicode.IsLetter(x), unicode.IsNumber(x):
			res.WriteRune(x)
		case x == '*':
			res.WriteString("_any_")
		case x == '>':
			res.WriteString("_all_")
		}
	}
	return res.String()
}

// consume runs the handler for every message, acks on done == true
// and asks for a delayed redelivery otherwise
func consume(c jetstream.Consumer, v func(topic string, body []byte) (done bool), double bool) (err error) {
	_, err = c.Consume(func(m jetstream.Msg) {
		if !v(m.Subject(), m.Data()) {
			m.NakWithDelay(NakDelay)
			return
		}
		if double {
			c, cancel := ctx()
			defer cancel()
			m.DoubleAck(c)
			return
		}
		m.Ack()
	})
	return
}

// publish sends a message through jetstream and waits for the server ack
func (a *Engine) jsPublish(subj string, b []byte) (err error) {
	if err = a.js(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()
	_, err = a.stream.Publish(c, subj, b)
	return
}

// int
func IntBytes(i int) (r []byte) {
	r = make([]byte, 8)
	binary.LittleEndian.PutUint64(r, uint64(i))
	return
}

// BytesInt decodes a value written by IntBytes, 0 for anything shorter than 8 bytes
func BytesInt(b []byte) (i int) {
	if len(b) < 8 {
		return 0
	}
	return int(binary.LittleEndian.Uint64(b))
}
