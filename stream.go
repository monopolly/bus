package bus

import (
	"errors"

	"github.com/nats-io/nats.go/jetstream"
)

// ErrNoSubjects is returned when a stream is created without subjects
var ErrNoSubjects = errors.New("bus: at least one subject is required")

// stream: every subscriber gets its own copy of each message,
// messages are kept while there is at least one interested consumer
type Stream struct {
	name   string
	stream jetstream.Stream
	conn   *Engine
}

// name and topics must
// signup, signup.*, signup.>
func (a *Engine) Stream(name string, subj ...string) (stream Stream, err error) {
	if len(subj) == 0 {
		err = ErrNoSubjects
		return
	}
	if err = a.js(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()

	s, err := a.stream.CreateOrUpdateStream(c, jetstream.StreamConfig{
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

func (a *Stream) ready() error {
	if a == nil || a.stream == nil || a.conn == nil {
		return ErrNoStream
	}
	return nil
}

// send task signup.ios, signup.email and wait for the server to persist it
func (a *Stream) Publish(subj string, b []byte) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	return a.conn.jsPublish(subj, b)
}

// signup.ios, signup.ios.>
// every Subscribe call gets its own (ephemeral) consumer and receives all messages.
// return false from v to get the message redelivered after NakDelay
func (a *Stream) Subscribe(subj string, v func(topic string, body []byte) (done bool)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()

	cons, err := a.stream.CreateConsumer(c, jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: subj,
	})
	if err != nil {
		return
	}
	return consume(cons, v, false)
}

func (a *Stream) Name() string {
	return a.name
}

func (a *Stream) JetStream() jetstream.Stream {
	return a.stream
}
