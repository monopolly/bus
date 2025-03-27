package bus

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"
)

// queue
type Queue struct {
	name   string
	stream jetstream.Stream
	conn   *Engine
}

// js.signup, js.signup.*, js.signup.>
func (a *Engine) Queue(name string, subj string, subjs ...string) (queue Queue, err error) {

	s, err := a.stream.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:      name,
		Subjects:  append([]string{subj}, subjs...),
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

// signup.ios, signup.ios.>
func (a *Queue) Group(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	c, err := a.stream.CreateOrUpdateConsumer(context.Background(), jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		Durable:       unique(subj),
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

// DoubleAck acknowledges a message and waits for ack reply from the server
func (a *Queue) GroupDoubleAck(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	c, err := a.stream.CreateOrUpdateConsumer(context.Background(), jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		Durable:       unique(subj),
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

func (a *Queue) Stream() jetstream.Stream {
	return a.stream
}
