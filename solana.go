package main

import (
	"context"
	"fmt"
	"github.com/portto/solana-go-sdk/client"
	_ "github.com/portto/solana-go-sdk/program/metaplex/tokenmeta"
	_ "github.com/portto/solana-go-sdk/rpc"
	"github.com/rs/zerolog/log"
	"go.uber.org/ratelimit"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

const (
	QUICKNODE    = ""
	RPCPOOL      = ""
	RPCPOOL_BETA = ""
	MOONRANK_RPC = ""
)

var DEVNET_ENDPOINTS = []string{"https://api.devnet.solana.com"}
var PUBLIC_ENDPOINTS = []string{"https://explorer-api.mainnet-beta.solana.com/",
	"https://mainnet.rpcpool.com/",
	"https://api.metaplex.solana.com/",
}
var PRIVATE_ENDPOINTS = []string{RPCPOOL, QUICKNODE, RPCPOOL_BETA}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetRateLimit() int {
	rv := 2 * len(PUBLIC_ENDPOINTS)

	if !UseDevNetEndpoints && !UsePublicEndpoints {
		rv = runtime.NumCPU() * 4
	}

	log.Printf("solana: rate limit %v\n", rv)
	return rv
}

func GetPrivateEndpoints() []string {
	return PRIVATE_ENDPOINTS
}

var RateLimiter = ratelimit.New(GetRateLimit(), ratelimit.Per(1*time.Second))

func CheckIfRateLimitErrorAndWait(err error) {
	x := fmt.Sprintf("%v", err)
	if strings.Contains(x, "429") {
		RateLimiter.Take()
	}
}

func GetEndpoints() (endpoints []string) {
	endpoints = GetPrivateEndpoints()
	if UseDevNetEndpoints {
		endpoints = DEVNET_ENDPOINTS
	} else if UsePublicEndpoints {
		endpoints = PUBLIC_ENDPOINTS
	}

	return
}

func GetTransaction(txSignature string) (tx *client.GetTransactionResponse, err error) {
	RateLimiter.Take()

	endpoints := GetEndpoints()

	for _, endpoint := range endpoints {
		c := client.NewClient(endpoint)

		log.Printf("GetTransaction(%v): %v\n", endpoint, txSignature)

		tx, err = c.GetTransaction(
			context.TODO(),
			txSignature)

		if err != nil {
			log.Printf("GetTransaction(%v): failed to get transaction `%v`, err: %v\n", endpoint, txSignature, err)
			CheckIfRateLimitErrorAndWait(err)
		} else {
			return
		}
	}

	return nil, err
}

func GetAccountInfo(address string) (accountInfo client.AccountInfo) {
	endpoints := GetEndpoints()

	for _, endpoint := range endpoints {
		if true {
			log.Printf("GetAccountInfo(%v): address %v", endpoint, address)
		}

		c := client.NewClient(endpoint)

		accountInfo, err := c.GetAccountInfo(
			context.TODO(),
			address,
		)
		if err != nil {
			log.Printf("GetAccountInfo(%v): failed to get account info, err: %v", endpoint, err)
		} else {
			return accountInfo
		}
	}

	return accountInfo
}
