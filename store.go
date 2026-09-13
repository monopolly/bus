package bus

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go/jetstream"
)

// ErrInvalidInt is returned by Int when the stored value is not an int written by AddInt
var ErrInvalidInt = errors.New("bus: value is not an int")

type Store struct {
	store jetstream.KeyValue
}

func (a *Store) ready() error {
	if a == nil || a.store == nil {
		return ErrNoStore
	}
	return nil
}

// KeyValue returns the underlying jetstream bucket
func (a *Store) KeyValue() jetstream.KeyValue {
	if a == nil {
		return nil
	}
	return a.store
}

// add mail.host, mail.token etc
func (a *Store) Add(k string, v []byte) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()
	_, err = a.store.Put(c, k, v)
	return
}

// add mail.host, mail.token etc
func (a *Store) AddString(k string, v string) (err error) {
	return a.Add(k, []byte(v))
}

// add mail.host, mail.token etc
func (a *Store) AddInt(k string, v int) (err error) {
	return a.Add(k, IntBytes(v))
}

// get mail.host, mail.token etc
func (a *Store) Get(k string) (res []byte, err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()
	v, err := a.store.Get(c, k)
	if err != nil {
		return
	}
	res = v.Value()
	return
}

// get mail.host, mail.token etc
func (a *Store) String(k string) (res string, err error) {
	b, err := a.Get(k)
	if err != nil {
		return
	}
	res = string(b)
	return
}

// get int
func (a *Store) Int(k string) (res int, err error) {
	b, err := a.Get(k)
	if err != nil {
		return
	}
	if len(b) != 8 {
		err = ErrInvalidInt
		return
	}
	res = BytesInt(b)
	return
}

// Exists reports whether the key is present
func (a *Store) Exists(k string) bool {
	_, err := a.Get(k)
	return err == nil
}

// delete key with its history
func (a *Store) Delete(k string) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()
	return a.store.Purge(c, k)
}

// keys, empty slice when the bucket is empty
func (a *Store) Keys() (res []string, err error) {
	if err = a.ready(); err != nil {
		return
	}
	c, cancel := ctx()
	defer cancel()
	res, err = a.store.Keys(c)
	if errors.Is(err, jetstream.ErrNoKeysFound) {
		return []string{}, nil
	}
	return
}

// updates monitoring mail.* mail.>
// f is called for every future change of a matching key,
// on delete newvalue is nil
func (a *Store) Watch(keys string, f func(k string, newvalue []byte)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	w, err := a.store.Watch(context.Background(), keys, jetstream.UpdatesOnly())
	if err != nil {
		return
	}
	go watch(w, f)
	return
}

// updates monitoring all keys
func (a *Store) Updates(f func(k string, newvalue []byte)) (err error) {
	if err = a.ready(); err != nil {
		return
	}
	w, err := a.store.WatchAll(context.Background(), jetstream.UpdatesOnly())
	if err != nil {
		return
	}
	go watch(w, f)
	return
}

// watch delivers entries until the watcher is stopped or the connection is closed
func watch(w jetstream.KeyWatcher, f func(k string, newvalue []byte)) {
	for e := range w.Updates() {
		// nil marks the end of initial values
		if e == nil {
			continue
		}
		f(e.Key(), e.Value())
	}
}
