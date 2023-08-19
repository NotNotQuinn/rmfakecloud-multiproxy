//go:generate go run generate/versioninfo.go

// rmfakecloud-multiproxy is a configurable reverse proxy to inject
// virtual cloud integrations and log network traffic
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func _main() error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	upstream, err := url.Parse(cfg.UpstreamCloudURL)
	if err != nil {
		return fmt.Errorf("invalid upstream address: %v", err)
	}

	srv := http.Server{
		Handler: &httputil.ReverseProxy{
			Rewrite: func(req *httputil.ProxyRequest) {
				// :443 is sometimes appended in 2.15? Never seen it on 3.5.2
				domain := strings.TrimSuffix(req.In.Host, ":443")
				requested_url := fmt.Sprintf("%s https://%s%s", req.In.Method, domain, req.In.URL)
				request_content, err := httputil.DumpRequest(req.Out, true)
				if err != nil {
					fmt.Printf("error dumping request %q: %v\n", requested_url, err)
					return
				}

				req.Out.URL.Scheme = "https"
				// Save this information to print later, because of async printing/buffer issues.
				req.Out = req.Out.WithContext(context.WithValue(
					context.Background(),
					"rmfakecloud.orig-request-str",
					fmt.Sprintf("%s\n%s", requested_url, request_content),
				))

				ip, err := resolve_host(domain)
				if err != nil {
					log.Println(err)
					log.Printf("Unable to resolve host %q\n", domain)
					return
				}
				req.Out.URL.Host = fmt.Sprint(ip)
			},
			// Ignore TLS verify, because we are accessing by IP address
			// remarkable's certs don't include ip records. """impossible""" to verify.
			// Unless you can figure out how to tell it that we know the domain name,
			// or integrate resolve_host(...) into the transport directly. DialContext?
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			ModifyResponse: func(r *http.Response) error {
				request_content := r.Request.Context().Value("rmfakecloud.orig-request-str").(string)
				response_content, err := httputil.DumpResponse(r, true)
				// All in one print statement to avoid async printing issues with many requests.
				if err != nil {
					fmt.Printf("===== Round Trip: %s\n===\nerror dumping response: %s\n===\n", request_content, err)
					return err
				} else {
					fmt.Printf("===== Round Trip: %s\n===\n%s\n===\n", request_content, response_content)
				}
				return nil
			},
		},
		Addr: cfg.ProxyListenAddr + ":443",
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

	log.Printf("cert-file=%s key-file=%s listen-addr=%s upstream-url=%s", cfg.TLSCertificateFile, cfg.TLSKeyFile, srv.Addr, upstream.String())
	if err := srv.ListenAndServeTLS(cfg.TLSCertificateFile, cfg.TLSKeyFile); err != http.ErrServerClosed {
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
