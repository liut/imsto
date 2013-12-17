package backend

import (
	"errors"
	"wpst.me/calf/config"
	"wpst.me/calf/db"
)

type FarmFunc func(string) (Wagoner, error)

type engine struct {
	name string
	farm FarmFunc
}

type Wagoner interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte, meta db.Hstore) (db.Hstore, error)
	Exists(key string) (bool, error)
	Del(key string) error
}

var engines = make(map[string]engine)

// Register a Engine
func RegisterEngine(name string, farm FarmFunc) {
	if farm == nil {
		panic("imsto: Register engine is nil")
	}
	if _, dup := engines[name]; dup {
		panic("imsto: Register called twice for engine " + name)
	}
	engines[name] = engine{name, farm}
}

// get a intance of Wagoner by a special config name
func FarmEngine(sn string) (Wagoner, error) {
	name := config.GetValue(sn, "engine")
	if engine, ok := engines[name]; ok {
		return engine.farm(sn)
	}

	return nil, errors.New("invalid engine name: " + name)
}