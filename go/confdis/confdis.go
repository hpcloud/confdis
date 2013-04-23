// confdis manages JSON based configuration in redis
package confdis

import (
	"encoding/json"
	"github.com/vmihailenco/redis"
	"net"
	"reflect"
)

type structCreator func() interface{}

type ConfDis struct {
	rootKey    string
	structType reflect.Type
	pubChannel string
	Config     interface{}
	redis      *redis.Client
	Changes    chan error // Channel to receive config updates (value is return of reload())
}

func New(addr, rootKey string, structVal interface{}) (*ConfDis, error) {
	c := ConfDis{
		rootKey,
		reflect.TypeOf(structVal),
		rootKey + ":_changes",
		createStruct(reflect.TypeOf(structVal)),
		nil,
		make(chan error)}
	if err := c.connect(addr); err != nil {
		return nil, err
	}
	if _, err := c.reload(); err != nil {
		return nil, err
	}
	return &c, c.watch()
}

func createStruct(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}

// Save saves current config onto redis. WARNING: Save() may not work
// correctly if there are concurrent changes from other clients
// (notified via pubsub); see reload() below.
func (c *ConfDis) Save() error {
	if data, err := json.Marshal(c.Config); err != nil {
		return err
	} else {
		if r := c.redis.Set(c.rootKey, string(data)); r.Err() != nil {
			return r.Err()
		}
		if r := c.redis.Publish(c.pubChannel, "confdis"); r.Err() != nil {
			return r.Err()
		}
	}
	return nil
}

func (c *ConfDis) connect(addr string) error {
	// Bug #97459 -- is the redis client library faking connection for
	// the down server?
	if conn, err := net.Dial("tcp", addr); err != nil {
		return err
	} else {
		conn.Close()
	}

	c.redis = redis.NewTCPClient(addr, "", 0)
	return nil
}

// reload reloads the config tree from redis.
func (c *ConfDis) reload() (interface{}, error) {
	if r := c.redis.Get(c.rootKey); r.Err() != nil {
		return nil, r.Err()
	} else {
		config2 := createStruct(c.structType)
		if err := json.Unmarshal([]byte(r.Val()), config2); err != nil {
			return nil, err
		}
		// FIXME: make it atomic in conjunction with the edits prior
		// to Save()
		config1 := c.Config
		c.Config = config2
		return config1, nil
	}
	panic("unreachable")
}

// watch watches for changes from other clients
func (c *ConfDis) watch() error {
	pubsub, err := c.redis.PubSubClient()
	if err != nil {
		return err
	}

	ch, err := pubsub.Subscribe(c.pubChannel)
	if err != nil {
		return err
	}

	go func() {
		for {
			<-ch
			_, err := c.reload()
			// TODO: pass old config (_) if necessary in the future.
			c.Changes <- err
		}
	}()

	return nil
}
