package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	namespace string = "promlens"
	subsystem string = "gmp_token_proxy"
)

var (
	// BuildTime is the time that this binary was built represented as a UNIX epoch
	BuildTime string
	// GitCommit is the git commit value and is expected to be set during build
	GitCommit string
	// GoVersion is the Golang runtime version
	GoVersion = runtime.Version()
	// OSVersion is the OS version (uname --kernel-release) and is expected to be set during build
	OSVersion string
	// StartTime is the start time of the exporter represented as a UNIX epoch
	StartTime = time.Now().Unix()
)
var (
	prefix = flag.String("prefix", "", "Prefix to be applied to the proxied path")
	proxy  = flag.String("proxy", "0.0.0.0:7777", "The endpoint of this proxy")
	remote = flag.String("remote", "0.0.0.0:9090", "host:path of the remote Prometheus server")
)
var (
	counterBuildTime = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "build_info",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "A metric with a constant '1' value labels by build time, git commit, OS and Go versions",
		}, []string{
			"build_time",
			"git_commit",
			"os_version",
			"go_version",
		},
	)
	proxiedStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "proxied_status",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "The total number of proxied requests that returned a status code",
		}, []string{
			"code",
		},
	)
	proxiedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "proxied_total",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "The total number of proxied requests",
		}, []string{},
	)
	proxiedError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "proxied_error",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "The total number of proxied requests that errored",
		}, []string{},
	)
	tokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "tokens_total",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "The total number of token requests",
		}, []string{},
	)
	tokensError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "tokens_error",
			Namespace: namespace,
			Subsystem: subsystem,
			Help:      "The total number of token requests failed",
		}, []string{},
	)
)
var (
	scopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}
)

// Mint is a type that represents an Oauth2 TokenSource and a cached Token
type Mint struct {
	TokenSource oauth2.TokenSource
	Token       *oauth2.Token
}

// NewMint is a function that creates a new Mint
// It creates a new TokenSource but doesn't initially mint a Token
func NewMint() (*Mint, error) {
	log.Print("Creating TokenSource")
	ts, err := google.DefaultTokenSource(context.Background(), scopes...)
	if err != nil {
		log.Print(err)
		return &Mint{}, err
	}

	log.Print("Returning Mint")
	return &Mint{
		TokenSource: ts,
	}, nil
}

// GetAccessToken is a method that returns an AccessToken
// It mints a new Token if needed
// Either because there is no Token
// Or because the Token has expired
func (m *Mint) GetAccessToken() (string, error) {
	// Assume the token is cached and not expired
	ok := true
	// If the token is nil
	if m.Token == nil {
		log.Print("No token cached")
		ok = false
	} else {
		// There is a token
		if m.Token.Expiry.Before(time.Now()) {
			// But it has expired
			log.Print("Cached Token has expired")
			ok = false
		}
	}
	// if not OK, then we need a new token
	if !ok {
		log.Print("Minting new Token")
		var err error
		tokensTotal.With(prometheus.Labels{}).Inc()
		m.Token, err = m.TokenSource.Token()
		if err != nil {
			tokensError.With(prometheus.Labels{}).Inc()
			msg := "unable to mint new Token"
			log.Print(msg)
			return "", fmt.Errorf(msg)
		}
	}

	// We have a cached, unexpired token
	return m.Token.AccessToken, nil
}

// handler is a function that returns the proxying handler
// It closes over Mint, host|path
func handler(m *Mint, remote, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, rqst *http.Request) {
		// Upgrade request
		rqst.URL.Scheme = "https"

		// Replace host with remote endpoint
		rqst.URL.Host = remote

		// Combine prefix with existing path
		// No "/" is necessary between paths
		revisedPath := fmt.Sprintf("%s%s", prefix, rqst.URL.Path)
		rqst.URL.Path = revisedPath

		// Google Managed Prometheus does not support `refresh` property in QueryString, remove it
		q := rqst.URL.Query()
		q.Del("refresh")
		rqst.URL.RawQuery = q.Encode()

		url := rqst.URL.String()
		pRqst, err := http.NewRequest(rqst.Method, url, rqst.Body)
		if err != nil {
			msg := "unable to create proxied request"
			log.Print(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		// Copy incoming headers
		pRqst.Header = rqst.Header
		// Add Authorization header if not already present
		if len(pRqst.Header.Values("Authorization")) != 0 {
			msg := "leaving unexpected Authorization header(s) on incoming request unchanged"
			log.Print(msg)
		} else {
			token, err := m.GetAccessToken()
			if err != nil {
				msg := "unable to obtain token"
				log.Print(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			pRqst.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		}

		// Create Transport
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client := &http.Client{
			Transport: transport,
		}

		// Invoke proxied request
		proxiedTotal.With(prometheus.Labels{}).Inc()
		pResp, err := client.Do(pRqst)
		if err != nil {
			proxiedError.With(prometheus.Labels{}).Inc()
			msg := "unable to execute proxied request"
			log.Print(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		defer pResp.Body.Close()

		proxiedStatus.With(prometheus.Labels{
			"code": strconv.Itoa(pResp.StatusCode),
		}).Inc()

		// Replicate the origin's headers
		func(dst, src http.Header) {
			for k, vv := range src {
				for _, v := range vv {
					dst.Add(k, v)
				}
			}
		}(w.Header(), pResp.Header)

		w.WriteHeader(pResp.StatusCode)
		io.Copy(w, pResp.Body)
	}
}
func main() {
	counterBuildTime.With(prometheus.Labels{
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"os_version": OSVersion,
		"go_version": GoVersion,
	}).Inc()

	flag.Parse()
	if *proxy == "" {
		log.Fatal("flag `--proxy` must be provided and non-nil")
	}
	if *remote == "" {
		log.Fatal("flag `--remote` must be provided and non-nil")
	}

	mint, err := NewMint()
	if err != nil {
		log.Fatal("unable to create new Mint")
	}

	mux := http.NewServeMux()

	// Otherwise proxy
	mux.HandleFunc("/", handler(mint, *remote, *prefix))

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Avoid serving /favico.ico
	// Doing so triggers the root handler and this results in tokens being minted
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})

	srv := &http.Server{
		Addr:    *proxy,
		Handler: mux,
	}

	log.Printf("Starting proxy [%s]", *proxy)
	log.Fatal(srv.ListenAndServe())
}
