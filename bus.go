package bus

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// message
type Message = nats.Msg
type Conn = nats.Conn

var (
	// ErrNotConnected is returned when the engine has no live connection
	ErrNotConnected = errors.New("bus: not connected")
	// ErrNoStore is returned when a store was not created
	ErrNoStore = errors.New("bus: store is not initialized")
	// ErrNoStream is returned when jetstream is not available
	ErrNoStream = errors.New("bus: jetstream is not available")

	// RequestTimeout is the default timeout for Request
	RequestTimeout = 30 * time.Second
	// APITimeout is the default timeout for jetstream api calls (streams, consumers, kv)
	APITimeout = 10 * time.Second
	// NakDelay is the delay before a failed (done == false) message is redelivered
	NakDelay = time.Second
)

// storename like "store", "app","settings" etc
func New(host, token, storename string) (a *Engine, err error) {
	a = &Engine{
		token:     token,
		host:      host,
		storename: storename,
	}
	if err = a.init(); err != nil {
		a.Close()
		return nil, err
	}
	return
}

type Engine struct {
	conn      *Conn
	token     string
	host      string
	storename string

	stream jetstream.JetStream

	store *Store
}

func (a *Engine) init() (err error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	}
	if a.token != "" {
		opts = append(opts, nats.Token(a.token))
	}

	a.conn, err = nats.Connect(a.host, opts...)
	if err != nil {
		return
	}

	a.stream, err = jetstream.New(a.conn)
	if err != nil {
		return
	}

	a.store, err = a.Store(a.storename)
	return
}

func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), APITimeout)
}

// ready checks that the connection exists
func (a *Engine) ready() error {
	if a == nil || a.conn == nil {
		return ErrNotConnected
	}
	return nil
}

// js checks that jetstream exists
func (a *Engine) js() error {
	if err := a.ready(); err != nil {
		return err
	}
	if a.stream == nil {
		return ErrNoStream
	}
	return nil
}

// close conn immediately, pending messages are dropped
func (a *Engine) Close() {
	if a == nil || a.conn == nil {
		return
	}
	a.conn.Close()
}

// Drain unsubscribes all subscriptions, waits for in-flight
// handlers to finish and then closes the connection
func (a *Engine) Drain() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.conn.Drain()
}

// nats connection
func (a *Engine) Conn() *Conn {
	if a == nil {
		return nil
	}
	return a.conn
}

// jetstream
func (a *Engine) JetStream() jetstream.JetStream {
	if a == nil {
		return nil
	}
	return a.stream
}

// time.*.east or time.us.>
func (a *Engine) Publish(to string, res []byte) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.conn.Publish(to, res)
}

// time.*.east or time.us.>
func (a *Engine) Request(to string, res []byte) (resp []byte, err error) {
	return a.RequestTimeout(to, res, RequestTimeout)
}

// time.*.east or time.us.>
func (a *Engine) RequestTimeout(to string, res []byte, timeout time.Duration) (resp []byte, err error) {
	if err = a.ready(); err != nil {
		return
	}
	m, err := a.conn.Request(to, res, timeout)
	if err != nil {
		return
	}
	resp = m.Data
	return
}

// time.*.east or time.us.>
func (a *Engine) Subscribe(to string, v func([]byte)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	_, err = a.conn.Subscribe(to, func(msg *nats.Msg) {
		v(msg.Data)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) SubscribeMessage(to string, v func(*Message)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	_, err = a.conn.Subscribe(to, func(msg *nats.Msg) {
		v(msg)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) Group(to, group string, v func([]byte)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	_, err = a.conn.QueueSubscribe(to, group, func(msg *nats.Msg) {
		v(msg.Data)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) GroupMessage(to, group string, v func(*Message)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	_, err = a.conn.QueueSubscribe(to, group, func(msg *nats.Msg) {
		v(msg)
	})
	return
}

// default store
func (a *Engine) DefaultStore() *Store {
	if a == nil {
		return nil
	}
	return a.store
}

// store
func (a *Engine) Add(k string, v []byte) (err error) {
	return a.DefaultStore().Add(k, v)
}

// store
func (a *Engine) AddString(k, v string) (err error) {
	return a.DefaultStore().AddString(k, v)
}

// store
func (a *Engine) AddInt(k string, v int) (err error) {
	return a.DefaultStore().AddInt(k, v)
}

// store
func (a *Engine) Get(k string) (res []byte, err error) {
	return a.DefaultStore().Get(k)
}

// store
func (a *Engine) String(k string) (res string, err error) {
	return a.DefaultStore().String(k)
}

// store
func (a *Engine) Int(k string) (res int, err error) {
	return a.DefaultStore().Int(k)
}

// Store creates (or opens) a persistent key-value bucket
func (a *Engine) Store(name string) (store *Store, err error) {
	return a.createStore(jetstream.KeyValueConfig{Bucket: name})
}

// CreateTTLStore creates (or opens) a persistent key-value bucket where
// every key expires after ttl
func (a *Engine) CreateTTLStore(name string, ttl time.Duration) (store *Store, err error) {
	return a.createStore(jetstream.KeyValueConfig{Bucket: name, TTL: ttl})
}

// CreateMemoryStore creates (or opens) an in-memory key-value bucket,
// optional ttl expires keys
func (a *Engine) CreateMemoryStore(name string, ttl ...time.Duration) (store *Store, err error) {
	cfg := jetstream.KeyValueConfig{Bucket: name, Storage: jetstream.MemoryStorage}
	if len(ttl) > 0 {
		cfg.TTL = ttl[0]
	}
	return a.createStore(cfg)
}

func (a *Engine) createStore(cfg jetstream.KeyValueConfig) (store *Store, err error) {
	if err = a.js(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()

	kv, err := a.stream.CreateOrUpdateKeyValue(c, cfg)
	if err != nil {
		return
	}
	store = &Store{store: kv}
	return
}
