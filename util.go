package main

import (
	"encoding/json"
	"github.com/portto/solana-go-sdk/common"
	"github.com/rs/zerolog/log"
	"strings"
)

func ValidatePubKey(pubkey string) bool {
	pk := common.PublicKeyFromString(pubkey)
	return pk.String() == pubkey
}

func CleanStringSlice(s []string) (x []string) {
	for _, p := range s {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			x = append(x, p)
		}
	}

	return x
}

type PublicKey struct {
	common.PublicKey
}

func (pk PublicKey) MarshalJSON() ([]byte, error) {
	log.Printf("MarshalJSON: pk %+v", pk)
	return json.Marshal(pk.String())
}
