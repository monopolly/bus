package bus

import (
	"github.com/nats-io/nats.go/jetstream"
)

// queue: every message is delivered to exactly one worker of the group
// and removed from the stream once acknowledged
type Queue struct {
	name   string
	stream jetstream.Stream
	conn   *Engine
}

// js.signup, js.signup.*, js.signup.>
func (a *Engine) Queue(name string, subj string, subjs ...string) (queue Queue, err error) {
	if err = a.js(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()

	s, err := a.stream.CreateOrUpdateStream(c, jetstream.StreamConfig{
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

func (a *Queue) ready() error {
	if a == nil || a.stream == nil || a.conn == nil {
		return ErrNoStream
	}
	return nil
}

// Publish sends a task to the queue and waits for the server to persist it
func (a *Queue) Publish(subj string, b []byte) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	return a.conn.jsPublish(subj, b)
}

// signup.ios, signup.ios.>
// all Group calls with the same subj share one durable consumer,
// so messages are load balanced between them.
// return false from v to get the message redelivered after NakDelay
func (a *Queue) Group(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	return a.group(subj, v, false)
}

// GroupDoubleAck acknowledges a message and waits for ack reply from the server
func (a *Queue) GroupDoubleAck(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	return a.group(subj, v, true)
}

func (a *Queue) group(subj string, v func(topic string, body []byte) (done bool), double bool) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()

	cons, err := a.stream.CreateOrUpdateConsumer(c, jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		Durable:       unique(subj),
		FilterSubject: subj,
	})
	if err != nil {
		return
	}
	return consume(cons, v, double)
}

func (a *Queue) Name() string {
	return a.name
}

func (a *Queue) Stream() jetstream.Stream {
	return a.stream
}
