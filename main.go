//go:generate go run generate/versioninfo.go

// rmfakecloud-multiproxy is a configurable reverse proxy to inject
// virtual cloud integrations and log network traffic
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// To be called in Rewrite()
func logHTTP_in_Rewrite(domain string, req *httputil.ProxyRequest) {
	requested_url := fmt.Sprintf("%s https://%s%s", req.In.Method, domain, req.In.URL)
	request_content, err := httputil.DumpRequest(req.Out, true)
	if err != nil {
		fmt.Printf("error dumping request %q: %v\n", requested_url, err)
		return
	}

	// Save this information to print later, because of async printing/buffer issues.
	req.Out = req.Out.WithContext(context.WithValue(
		context.Background(),
		httpLog{},
		fmt.Sprintf("%s\n%s", requested_url, request_content),
	))
}

func logHTTP_in_ModifyResponse(r *http.Response) {
	request_content := r.Request.Context().Value(httpLog{}).(string)
	response_content, err := httputil.DumpResponse(r, true)
	// All in one print statement to avoid async printing issues with many requests.
	if err != nil {
		fmt.Printf("===== Round Trip: %s\n===\nerror dumping response: %s\n===\n", request_content, err)
	} else {
		fmt.Printf("===== Round Trip: %s\n===\n%s\n===\n", request_content, response_content)
	}
}

func Rewrite(cfg *ConfigFile, req *httputil.ProxyRequest) {
	// :443 is sometimes appended in 2.15? Never seen it on 3.5.2
	domain := strings.TrimSuffix(req.In.Host, ":443")
	if cfg.IsSet("LOG_HTTP_REQUESTS") {
		logHTTP_in_Rewrite(domain, req)
	}

	req.Out.URL.Scheme = "https"
	ip, err := resolve_host(domain)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("Unable to resolve host %q\n", domain)
		return
	}
	req.Out.URL.Host = fmt.Sprint(ip)
}

func ModifyResponse(cfg *ConfigFile, r *http.Response) error {
	if cfg.IsSet("LOG_HTTP_REQUESTS") {
		logHTTP_in_ModifyResponse(r)
	}
	return nil
}

// A context key to store a formatted string of an HTTP request.
type httpLog struct{}

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
				Rewrite(cfg, req)
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
