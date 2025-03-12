package bus

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type Stream = jetstream.Stream

// name and topics must
// signup, signup.*, signup.>
func (a *Engine) Queue(name string, subj ...string) (queue Queue, err error) {
	if subj == nil {
		err = fmt.Errorf("subj must be")
		return
	}
	s, err := a.stream.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:      name,
		Subjects:  subj,
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return
	}
	queue.stream = s
	queue.name = name
	queue.conn = a
	return
}

// queue
type Queue struct {
	name   string
	stream jetstream.Stream
	conn   *Engine
}

// send queue task signup.ios, signup.email
func (a *Queue) Publish(subj string, b []byte) (err error) {
	Wait(a.conn)
	return a.conn.Publish(subj, b)
}

// signup.ios, signup.ios.>
func (a *Queue) Subscribe(group, subj string, v func(topic string, body []byte) (done bool)) (err error) {
	c, err := a.stream.CreateConsumer(context.Background(), jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		Durable:       group,
		FilterSubject: subj,
	})
	if err != nil {
		return
	}

	go c.Consume(func(m jetstream.Msg) {
		done := v(m.Subject(), m.Data())
		switch done {
		case true:
			m.DoubleAck(context.Background())
		default:
			m.Nak()
		}
	})

	return
}
