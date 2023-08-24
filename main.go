//go:generate go run generate/versioninfo.go

// rmfakecloud-multiproxy is a configurable reverse proxy to inject
// virtual cloud integrations and log network traffic
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/notnotquinn/rmfakecloud-multiproxy/intercept/network"
)

// To be called in Rewrite()
func logHTTP_in_Rewrite(outgoingHost string, req *httputil.ProxyRequest) {
	incoming_req := fmt.Sprintf("%s https://%s%s", req.In.Method, req.In.Host, req.In.URL.RequestURI())
	outgoing_req := fmt.Sprintf("%s https://%s%s", req.Out.Method, outgoingHost, req.Out.URL.RequestURI())
	request_dump, err := httputil.DumpRequest(req.Out, true)
	if err != nil {
		fmt.Printf("error dumping request %q: %v\n", incoming_req, err)
		return
	}

	// Remove Accept-Encoding (eg. gzip, deflate)
	// Otherwise we would need to decode and write our own DumpRequest() func.
	// The Go http implementation may add their own 'Accept-Encoding: gzip' header. But if so, it
	// is decoded transparently when reading resp.Body. (see http.Transport#DisableCompression)
	req.Out.Header.Del("Accept-Encoding")

	// Save this information to print later, because of async printing/buffer issues.
	req.Out = req.Out.WithContext(context.WithValue(
		context.Background(),
		httpLogContextKey{},
		httpLog{
			incoming_req: incoming_req,
			outgoing_req: outgoing_req,
			request_dump: string(request_dump),
		},
	))
}

func dumpResponse(r *http.Response, adverb string) string {
	response_dump, err := httputil.DumpResponse(r, true)

	if err != nil {
		return fmt.Sprintf("error dumping %s response for: %s", adverb, err)
	} else {
		return string(response_dump)
	}
}

func printHttpLog(r *http.Response, unmodified_resp string, modified bool) {
	dump := r.Request.Context().Value(httpLogContextKey{}).(httpLog)

	// All in one print statement to avoid async printing issues with many requests.
	msg := "------ Round Trip ------\n"
	msg += "<=== incoming " + dump.incoming_req + "\n"
	msg += "===> outgoing " + dump.outgoing_req + "\n"
	msg += dump.request_dump + "\n"
	msg += "=== Original Server Response"
	if !modified {
		msg += " (wasn't modified)"
	}
	msg += "\n" + unmodified_resp + "\n"
	if modified {
		msg += "=== Modified Response\n"
		msg += dumpResponse(r, "modified") + "\n"
	}
	msg += "===\n"
	fmt.Print(msg)
}

func Rewrite(cfg *ConfigFile, upstream *url.URL, req *httputil.ProxyRequest) {
	outgoingHost := ""

	if cfg.IsSet("USE_OFFICIAL_CLOUD") {
		outgoingHost = strings.TrimSuffix(req.In.Host, ":443")
		req.Out.URL.Scheme = "https"
		ip, err := resolve_host(outgoingHost)
		if err != nil {
			fmt.Println(err)
			fmt.Printf("Unable to resolve host %q\n", outgoingHost)
			return
		}
		req.Out.URL.Host = fmt.Sprint(ip)
		// req.Out.Header.Set("Host", outgoingHost)
	} else {
		outgoingHost = strings.TrimSuffix(upstream.Host, ":443")
		req.SetURL(upstream)
		req.Out.Host = upstream.Host
	}

	if cfg.IsSet("LOG_HTTP_REQUESTS") {
		logHTTP_in_Rewrite(outgoingHost, req)
	}
}

func ModifyResponse(cfg *ConfigFile, r *http.Response) error {
	var modified bool = false
	if cfg.IsSet("LOG_HTTP_REQUESTS") {
		unmodified_resp := dumpResponse(r, "unmodified")
		// Capture variables by reference
		defer func() { printHttpLog(r, unmodified_resp, modified) }()
	}
	if r.Request.Method == "GET" && r.Request.URL.Path == "/integrations/v1/" && r.StatusCode == 200 {
		modified = true
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		var parsedResp network.GetIntegrationsResp
		if err := json.Unmarshal(body, &parsedResp); err != nil {
			return fmt.Errorf("unable to unmarshal integration: %w", err)
		}
		if parsedResp.Integrations == nil {
			parsedResp.Integrations = []network.Integration{}
		}
		parsedResp.Integrations = append(parsedResp.Integrations, network.Integration{
			ID:         "onepiece",
			UserID:     "guest-onepiece",
			Name:       "One Piece",
			Added:      time.Now(),
			ProviderID: "virtual-integration:onepiece",
			Issues:     []any{},
		})
		encodedResp, err := json.Marshal(parsedResp)
		if err != nil {
			return fmt.Errorf("unable to marshal integration: %w", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(encodedResp))
		r.Header.Set("Content-Length", fmt.Sprint(len(encodedResp)))
		r.ContentLength = int64(len(encodedResp))
	}
	return nil
}

// A context key to store a formatted HTTP request.
//
// Stores httpLog{...} struct
type httpLogContextKey struct{}
type httpLog struct {
	incoming_req string
	outgoing_req string
	request_dump string
}

func _main() error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	upstream, err := url.Parse(cfg.Get("UPSTREAM_CLOUD_URL"))
	if err != nil {
		return fmt.Errorf("invalid upstream address: %v", err)
	}

	srv := http.Server{
		Handler: &httputil.ReverseProxy{
			Rewrite: func(req *httputil.ProxyRequest) {
				Rewrite(cfg, upstream, req)
			},
			// Ignore TLS verify, because we are accessing by IP address
			// remarkable's certs don't include ip records. """impossible""" to verify.
			// Unless you can figure out how to tell it that we know the domain name,
			// or integrate resolve_host(...) into the transport directly. DialContext?
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			ModifyResponse: func(r *http.Response) error {
				return ModifyResponse(cfg, r)
			},
		},
		Addr: cfg.Get("PROXY_LISTEN_ADDR") + ":443",
	}

	done := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		fmt.Println(<-sig)

		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Printf("Shutdown: %v", err)
		}
		close(done)
	}()

	fmt.Printf("Configuration (raw):\n")
	for _, opt := range validOptions {
		fmt.Printf("  %s=%s\n", opt.Name, cfg.Get(opt.Name))
	}
	fmt.Printf("Configuration:\n")
	fmt.Printf("  srv.Addr: %v\n", srv.Addr)
	fmt.Printf("  upstream.String(): %v\n", upstream.String())

	fmt.Printf("Active modes:\n")
	if cfg.IsSet("USE_OFFICIAL_CLOUD") {
		fmt.Printf("  upstream = <official cloud>\n")
	} else {
		fmt.Printf("  upstream = %s\n", cfg.Get("UPSTREAM_CLOUD_URL"))
	}
	if cfg.IsSet("LOG_HTTP_REQUESTS") {
		fmt.Printf("  Log HTTP Requests\n")
	}

	certFile := cfg.Get("TLS_CERTIFICATE_FILE")
	keyFile := cfg.Get("TLS_KEY_FILE")

	if err := srv.ListenAndServeTLS(certFile, keyFile); err != http.ErrServerClosed {
		return fmt.Errorf("ListenAndServeTLS: %v", err)
	}

	<-done
	return nil
}

func main() {
	err := _main()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
