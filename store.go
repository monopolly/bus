package bus

import (
	"context"

	"github.com/monopolly/numbers"
	"github.com/nats-io/nats.go/jetstream"
)

type Store struct {
	store jetstream.KeyValue
}

// add mail.host, mail.token etc
func (a *Store) Add(k string, v []byte) (err error) {
	_, err = a.store.Put(context.Background(), k, v)
	return
}

// add mail.host, mail.token etc
func (a *Store) AddString(k string, v string) (err error) {
	_, err = a.store.PutString(context.Background(), k, v)
	return
}

// add mail.host, mail.token etc
func (a *Store) AddInt(k string, v int) (err error) {
	_, err = a.store.Put(context.Background(), k, numbers.IntBytes(v))
	return
}

// get mail.host, mail.token etc
func (a *Store) Get(k string) (res []byte, err error) {
	v, err := a.store.Get(context.Background(), k)
	if err != nil {
		return
	}

	res = v.Value()
	return
}

// get mail.host, mail.token etc
func (a *Store) String(k string) (res string, err error) {
	v, err := a.store.Get(context.Background(), k)
	if err != nil {
		return
	}

	res = string(v.Value())
	return
}

// delete key
func (a *Store) Delete(k string) (err error) {
	return a.store.Purge(context.Background(), k)
}

// keys
func (a *Store) Keys() (res []string, err error) {
	return a.store.Keys(context.Background())
}

// updates monitoring mail.* mail.>
func (a *Store) Watch(keys string, f func(k string, newvalue []byte)) {
	w, _ := a.store.Watch(context.Background(), keys)
	go func() {
		b := <-w.Updates()
		f(b.Key(), b.Value())
	}()
}

// updates monitoring all keys
func (a *Store) Updates(f func(k string, newvalue []byte)) {
	w, _ := a.store.WatchAll(context.Background())
	go func() {
		b := <-w.Updates()
		f(b.Key(), b.Value())
	}()
}

// get int
func (a *Store) Int(k string) (res int, err error) {
	v, err := a.store.Get(context.Background(), k)
	if err != nil {
		return
	}

	res = numbers.BytesInt(v.Value())
	return
}
