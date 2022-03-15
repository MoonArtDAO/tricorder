package main

import (
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type TTLMap struct {
	sync.Map
}

type TTLMapEntry struct {
	Value  interface{}
	IsSet  bool
	Expire time.Time
}

func (m *TTLMap) Store(key string, value interface{}, isSet bool, ttl time.Duration) {
	expire := time.Now().Add(ttl)
	log.Printf("TTLMap.Store: key %v isSet %v ttl %v, expire %v", key, isSet, ttl, expire)
	m.Map.Store(key, TTLMapEntry{Value: value, IsSet: isSet, Expire: expire})
}

func (m *TTLMap) Load(key interface{}) (value interface{}, isSet bool, ok bool) {
	log.Printf("TTLMap.Load: key %v", key)
	value, ok = m.Map.Load(key)
	if !ok {
		return value, false, ok
	}

	e := value.(TTLMapEntry)
	if time.Now().After(e.Expire) {
		log.Printf("TTLMap.Load: key %v expired", key)
		m.Map.Delete(key)
		return nil, false, false
	}

	return e.Value, e.IsSet, ok
}

func (m *TTLMap) LoadOrStore(key, value interface{}, isSet bool, ttl time.Duration) (actual interface{}, loaded bool) {
	expire := time.Now().Add(ttl)
	log.Printf("TTLMap.LoadOrStore: key %v isSet %v expire %v", key, isSet, expire)
	return m.Map.LoadOrStore(key, TTLMapEntry{Value: value, IsSet: isSet, Expire: expire})
}

func (m *TTLMap) Delete(key interface{}) {
	log.Printf("TTLMap.Delete: key %v", key)
	m.Map.Delete(key)
}

func (m *TTLMap) ExpireBefore(t time.Time) {
	deleted := 0
	m.Map.Range(func(key, value interface{}) bool {
		e := value.(TTLMapEntry)
		if e.Expire.Before(t) {
			log.Printf("TTLMap.ExpireBefore: key %v expired", key)
			m.Map.Delete(key)
			deleted++
		}

		return true
	})

	log.Printf("TTLMap.ExpireBefore: deleted %v", deleted)
}
