package main

import (
	"github.com/portto/solana-go-sdk/client"
	"github.com/rs/zerolog/log"
	"time"
)

type State struct {
	TxMap TTLMap
	TxIL  Interlock
}

var CurrentState State

func GetCachedTx(txSignature string) (tx *client.GetTransactionResponse, isSet bool, err error) {
	cachedTx, isSet, ok := CurrentState.TxMap.Load(txSignature)
	log.Printf("GetCachedTx: txSignature %v was cached %v isSet %v\n", txSignature, ok, isSet)
	if ok {
		tx = cachedTx.(*client.GetTransactionResponse)
	} else {
		tx, err = GetTransaction(txSignature)
		if err == nil {
			CurrentState.TxMap.Store(txSignature, tx, true, TX_CACHE_MAX_AGE)
			isSet = true
		} else {
			CurrentState.TxMap.Store(txSignature, tx, false, 1*time.Minute)
		}
	}

	return
}

func InterlockedGetCachedTx(txSignature string) (tx *client.GetTransactionResponse, isSet bool, err error) {
	ticket := CurrentState.TxIL.WaitOrStart(txSignature)
	log.Printf("InterlockedGetCachedTx: txSignature %v ticket %v\n", txSignature, ticket)
	if ticket > 0 {
		defer CurrentState.TxIL.ClearWait(txSignature, ticket)
	}

	return GetCachedTx(txSignature)
}
