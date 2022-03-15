package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const YIELD_TIME = 100 * time.Microsecond

type Interlock struct {
	IL sync.Map
}

func (il *Interlock) WaitOrStart(key string) uint64 {
	ticket := rand.Uint64()
	currentTicket, loaded := il.IL.LoadOrStore(key, ticket)
	log.Printf("Interlock#WaitOrStart: key %v ticket %v currentTicket %v loaded %v", key, ticket, currentTicket, loaded)

	if !loaded {
		log.Printf("Interlock#WaitOrStart: key %v ticket %v won", key, ticket)
		return ticket
	}

	for loaded {
		_, loaded = il.IL.Load(key)
		log.Printf("Interlock#WaitOrStart: key %v ticket %v yielding", key, ticket)
		time.Sleep(YIELD_TIME)
	}

	log.Printf("Interlock#WaitOrStart: key %v ticket %v exit", key, ticket)
	return 0
}

func (il *Interlock) ClearWait(key string, ticket uint64) {
	val, ok := il.IL.Load(key)
	if ok {
		val := val.(uint64)
		if val == ticket {
			log.Printf("Interlock#ClearWait: key %v ticket %v cleared", key, ticket)
			il.IL.Delete(key)
		}
	}
}
