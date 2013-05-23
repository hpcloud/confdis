// confdis manages JSON based configuration in redis
package confdis

import (
	"encoding/json"
	"fmt"
	"github.com/vmihailenco/redis"
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
	redis      *redis.Client
	Changes    chan error // Channel to receive config updates (value is return of reload())
}

const PUB_SUFFIX = ":_changes"

func New(client *redis.Client, rootKey string, structVal interface{}) (*ConfDis, error) {
	c := ConfDis{}
	c.redis = client
	c.rootKey = rootKey
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
		if r := c.redis.Set(c.rootKey, string(data)); r.Err() != nil {
			return r.Err()
		}
		if r := c.redis.Publish(c.rootKey+PUB_SUFFIX, "confdis"); r.Err() != nil {
			return r.Err()
		}
	}
	return nil
}

// reload reloads the config tree from redis.
func (c *ConfDis) reload() (interface{}, bool, error) {
	if r := c.redis.Get(c.rootKey); r.Err() != nil {
		return nil, true, r.Err()
	} else {
		config2 := createStruct(c.structType)
		if err := json.Unmarshal([]byte(r.Val()), config2); err != nil {
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
	pubsub, err := c.redis.PubSubClient()
	if err != nil {
		return err
	}

	ch, err := pubsub.Subscribe(c.rootKey + PUB_SUFFIX)
	if err != nil {
		return err
	}

	go func() {
		for {
			<-ch
			_, _, err := c.reload()
			// TODO: pass old config (_) if necessary in the future.
			c.Changes <- err
		}
	}()

	return nil
}
