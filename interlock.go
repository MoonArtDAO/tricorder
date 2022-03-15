package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const MIN_YIELD_TIME = 100 // * time.Microsecond
const MAX_YIELD_TIME = 200 // * time.Microsecond

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
		n := time.Microsecond * time.Duration(MIN_YIELD_TIME+rand.Intn(MAX_YIELD_TIME-MIN_YIELD_TIME+1))
		log.Printf("Interlock#WaitOrStart: key %v ticket %v yielding for %v", key, ticket, n)
		time.Sleep(n)
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
