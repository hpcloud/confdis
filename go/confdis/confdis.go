// confdis manages JSON based configuration in redis
package confdis

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/store"
	"github.com/coreos/go-etcd/etcd"
	"reflect"
	"sync"
)

type structCreator func() interface{}

type ConfDis struct {
	rootKey    string
	structType reflect.Type
	config     interface{} // Read-only view of current config tree.
	rev        int64
	mux        sync.Mutex // Mutex to protect changes to config and rev.
	client     *etcd.Client
	Changes    chan error // Channel to receive config updates (value is return of reload())
}

func New(client *etcd.Client, rootKey string, structVal interface{}) (*ConfDis, error) {
	c := ConfDis{}
	c.client = client
	c.rootKey = fmt.Sprintf("/config/%s", rootKey)
	c.structType = reflect.TypeOf(structVal)
	c.config = createStruct(c.structType)
	c.Changes = make(chan error)
	if _, empty, err := c.reload(); err != nil {
		// Ignore if config doesn't already exist; it can be created
		// later.
		if !empty {
			return nil, err
		}
	}
	return &c, c.watch()
}

// GetConfig returns the current snapshot of config struct.
func (c *ConfDis) GetConfig() interface{} {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.config
}

// MustReceiveChanges listens for change notifications and updates the
// internal config. Will panic if there is an error reading the new
// config.
func (c *ConfDis) MustReceiveChanges() {
	for err := range c.Changes {
		if err != nil {
			panic(err)
		}
	}
}

// AtomicSave is like save, but only writes the changed config back to
// redis if somebody else did not make a change already (notified via
// pubsub). Note that the converse is not necessarily true; somebody
// else -- specifically, reload() -- *could* overwrite the changes
// written by AtomicSave.
func (c *ConfDis) AtomicSave(editFn func(interface{}) error) error {
	c.mux.Lock()
	previousconfig := c.config
	previousRev := c.rev
	c.mux.Unlock()

	if err := editFn(previousconfig); err != nil {
		return err
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	// Was config changed interim by reload()?
	if c.rev != previousRev {
		return fmt.Errorf(
			"config already changed (rev %d -> %d)", previousRev, c.rev)
	}
	if err := c.save(); err != nil {
		return err
	}
	c.rev += 1
	return nil
}

func createStruct(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}

func (c *ConfDis) save() error {
	if data, err := json.Marshal(c.config); err != nil {
		return err
	} else {
		if _, err := c.client.Set(c.rootKey, string(data), 0); err != nil {
			return err
		}
	}
	return nil
}

// reload reloads the config tree from redis.
func (c *ConfDis) reload() (interface{}, bool, error) {
	if results, err := c.client.Get(c.rootKey); err != nil {
		return nil, true, err
	} else {
		r := results[0]
		config2 := createStruct(c.structType)
		if err := json.Unmarshal([]byte(r.Value), config2); err != nil {
			return nil, false, err
		}
		c.mux.Lock()
		defer c.mux.Unlock()
		config1 := c.config
		c.config = config2
		c.rev += 1
		return config1, false, nil
	}
	panic("unreachable")
}

// watch watches for changes from other clients
func (c *ConfDis) watch() error {
	ch := make(chan *store.Response)

	go func() {
		rev := uint64(0) // TODO
		_, err := c.client.Watch(c.rootKey, rev, ch, nil)
		panic(err)
	}()

	go func() {
		for {
			<-ch
			// TODO: pass the value from <-ch to reload, so we don't
			// have to read again (potentially incorrect rev).
			_, _, err := c.reload()
			// TODO: pass old config (_) if necessary in the future.
			c.Changes <- err
		}
	}()

	return nil
}
