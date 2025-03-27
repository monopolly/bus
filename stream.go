package bus

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// queue
type Stream struct {
	name   string
	stream jetstream.Stream
	conn   *Engine
}

// name and topics must
// signup, signup.*, signup.>
func (a *Engine) Stream(name string, subj ...string) (stream Stream, err error) {
	if subj == nil {
		err = fmt.Errorf("subj must be")
		return
	}
	s, err := a.stream.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:      name,
		Subjects:  subj,
		Retention: jetstream.InterestPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return
	}
	stream.stream = s
	stream.name = name
	stream.conn = a
	return
}

// send queue task signup.ios, signup.email
func (a *Stream) Publish(subj string, b []byte) (err error) {
	Wait(a.conn)
	return a.conn.Publish(subj, b)
}

// signup.ios, signup.ios.>
func (a *Stream) Subscribe(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	c, err := a.stream.CreateConsumer(context.Background(), jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: subj,
	})
	if err != nil {
		return
	}

	go c.Consume(func(m jetstream.Msg) {
		done := v(m.Subject(), m.Data())
		switch done {
		case true:
			m.Ack()
		default:
			m.Nak()
		}
	})

	return
}
