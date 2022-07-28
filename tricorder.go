package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/justinas/alice"
	"github.com/portto/solana-go-sdk/client"
	_ "github.com/portto/solana-go-sdk/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	SERVICE          = "tricorder"
	DEBUG            = false
	TX_CACHE_MAX_AGE = 60 * time.Second
)

var ListenAddress string
var UseDevNetEndpoints bool
var UsePublicEndpoints bool

func InitFlags() {
	flag.StringVar(&ListenAddress, "addr", "localhost:8080", "listen address")
	flag.BoolVar(&UseDevNetEndpoints, "d", false, "use devnet endpoints")
	flag.BoolVar(&UsePublicEndpoints, "p", true, "use public endpoints")
	flag.Parse()
}

func IsValidTxSignature(txSignature string) bool {
	return true
}

type APITxResponse struct {
	Status string                         `json:"status"`
	Result *client.GetTransactionResponse `json:"result"`
	Error  *APIError                      `json:"error"`
}

func HandleTxV0(txSignature string, w http.ResponseWriter, r *http.Request) (err error, notFound bool) {
	if !IsValidTxSignature(txSignature) {
		return errors.New("not found"), true
	}

	tx, isSet, err := InterlockedGetCachedTx(txSignature)
	if err != nil {
		log.Error().Err(err).Msg("")
		return err, false
	} else if !isSet {
		return errors.New("yikes"), false
	}

	log.Printf("tx = %+v", tx)

	w.Header().Set("cache-control", "public, max-age=60")
	w.Header().Set("access-control-allow-origin", "*")
	w.Header().Set("content-type", "application/json")

	response := APITxResponse{
		Status: "ok",
		Result: tx,
	}

	json.NewEncoder(w).Encode(response)
	return nil, false
}

type APIError struct {
	ErrorID      string `json:"error_id"`
	ErrorMessage string `json:"error_mesage"`
}

type APIErrorResponse struct {
	Status string   `json:"status"`
	Error  APIError `json:"error"`
}

func MakeAPIInvalidVersionError(inputVersion string) (apiErrorResponse APIErrorResponse) {
	apiErrorResponse.Status = "fail"
	apiErrorResponse.Error.ErrorID = "invalid_version"
	apiErrorResponse.Error.ErrorMessage = fmt.Sprintf("`%v` is not a supported version. Valid versions are: `latest`, `0`.",
		inputVersion)

	return
}

var ValidVersions = map[string]bool{
	"0":      true,
	"latest": true,
}

func IsValidVersion(responseVersion string) bool {
	return ValidVersions[responseVersion]
}

func HandleTx(responseVersion string, txSignature string,
	w http.ResponseWriter, r *http.Request) (err error, notFound bool) {
	if !IsValidTxSignature(txSignature) {
		return errors.New("not found"), true
	}

	log.Printf("HandleTx: responseVersion %v, txSignature %v", responseVersion, txSignature)
	switch responseVersion {
	case "0":
		return HandleTxV0(txSignature, w, r)
		break
	case "latest":
		return HandleTxV0(txSignature, w, r)
		break
	}

	return RespondWithInvalidVersionError(w, r, responseVersion)
}

func RespondWithInvalidVersionError(w http.ResponseWriter, r *http.Request, responseVersion string) (err error, notFound bool) {
	m := MakeAPIInvalidVersionError(responseVersion)
	w.Header().Set("cache-control", "public, max-age=600")
	w.Header().Set("access-control-allow-origin", "*")
	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(m)

	return nil, false
}

func SetGlobalHeaders(w http.ResponseWriter) {
	if !DEBUG {
		w.Header().Set("strict-transport-security", "max-age=31536000; preload")
	}

	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("x-moon-art", SERVICE)
}

func SendNoCache(w http.ResponseWriter) {
	w.Header().Set("cache-control", "private, max-age=0, no-store, no-cache, must-revalidate")
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	SetGlobalHeaders(w)

	switch r.URL.Path {
	case "/":
		io.WriteString(w, SERVICE)
	default:
		handled := false
		notFound := true
		var err error

		parts := CleanStringSlice(strings.Split(r.URL.Path, "/"))
		log.Printf("IndexHandler: parts: %v", parts)
		if len(parts) > 2 {
			version := parts[0]
			object := parts[1]

			if !IsValidVersion(version) {
				err, notFound = RespondWithInvalidVersionError(w, r, version)
				if err == nil {
					handled = true
				}
			} else {
				switch object {
				case "tx":
					if len(parts) > 2 {
						txSig := parts[2]

						err, notFound = HandleTx(version, txSig, w, r)
						if err == nil {
							handled = true
						}
					}
				}
			}
		}

		if !handled {
			if err != nil {
				SendNoCache(w)
				log.Error().Err(err)
				http.Error(w, "503 service unavailable", http.StatusServiceUnavailable)
			} else if notFound {
				http.NotFound(w, r)
			}
		}
	}
}

func ConnectingIPHandler(fieldKey, header string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if val := r.Header.Get(header); val != "" {
				log := zerolog.Ctx(r.Context())
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(fieldKey, val)
				})
			} else if r.RemoteAddr != "" {
				log := zerolog.Ctx(r.Context())
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(fieldKey, r.RemoteAddr)
				})
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)
	InitFlags()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Timestamp().
		Str("role", SERVICE).
		Logger()

	log.Logger = logger

	log.Printf("starting up on %v", runtime.Version())

	c := alice.New()
	c = c.Append(hlog.NewHandler(logger))
	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	c = c.Append(ConnectingIPHandler("ip", "cf-connecting-ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.CustomHeaderHandler("ray", "cf-ray"))
	h := c.Then(http.HandlerFunc(IndexHandler))

	go MaintainCache()

	http.Handle("/", h)
	for {
		log.Printf("about to listen on %v", ListenAddress)
		http.ListenAndServe(ListenAddress, nil)
	}
}
