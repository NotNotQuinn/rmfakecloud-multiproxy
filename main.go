//go:generate go run generate/versioninfo.go

// rmfake-proxy is a fork of "secure", which is a super simple TLS termination proxy
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CertFile string `yaml:"cert"`
	KeyFile  string `yaml:"key"`
	Upstream string `yaml:"upstream"`
	Addr     string `yaml:"addr"`
}

var (
	version    bool
	configFile string
)

func getConfig() (config *Config, err error) {
	cfg := Config{}
	flag.StringVar(&configFile, "c", "", "config file")
	flag.StringVar(&cfg.Addr, "addr", ":443", "listen address")
	flag.StringVar(&cfg.CertFile, "cert", "", "path to cert file")
	flag.StringVar(&cfg.KeyFile, "key", "", "path to key file")
	flag.BoolVar(&version, "version", false, "print version string and exit")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"usage: %s -c [config.yml] [-addr host:port] -cert certfile -key keyfile [-version]\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if version {
		fmt.Fprintln(flag.CommandLine.Output(), Version)
		os.Exit(0)
	}

	if configFile != "" {
		var data []byte
		data, err = os.ReadFile(configFile)

		if err != nil {
			return
		}
		err = yaml.Unmarshal(data, &cfg)
		if err != nil {
			return nil, fmt.Errorf("cant parse config, %v", err)
		}
		if _, err := strconv.Atoi(cfg.Addr); err == nil {
			cfg.Addr = ":" + cfg.Addr

		}
		return &cfg, nil
	}

	return &cfg, nil
}

type dnsCacheEntry struct {
	expires time.Time
	ip      net.IP
}

var dns_client = dns.Client{}
var dns_cache = map[string]dnsCacheEntry{}

// edited from https://stackoverflow.com/questions/30043248#31627459
//
// We can't resolve using the system because we patched /etc/hosts,
// so we will just get back 127.0.0.1 -> end up with recursion
func resolve_host(domain string) (*net.IP, error) {
	server := "1.1.1.1:53" // Cloudflare DNS

	// Check cache
	if entry, ok := dns_cache[domain]; ok {
		if time.Now().Before(entry.expires) {
			return &entry.ip, nil
		}
	}

	// Not cached, raw dns request
	m := dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	r, t, err := dns_client.Exchange(&m, server)
	if err != nil {
		return nil, fmt.Errorf("DNS: %q: error: %v", domain, err)
	}
	for _, ans := range r.Answer {
		A_record := ans.(*dns.A)

		// Implement some rudimentary caching
		ttl := time.Duration(A_record.Hdr.Ttl) * time.Second
		dns_cache[domain] = dnsCacheEntry{
			expires: time.Now().Add(ttl),
			ip:      A_record.A,
		}

		log.Printf("DNS: %q: %s (ttl %d)", domain, A_record.A, ttl)
		return &A_record.A, nil
	}

	return nil, fmt.Errorf("DNS: %q: no result in %v", domain, t)
}

// Formatted request as a string.
//
// Stores requested URL on the first line,
// and the request body from httputil.DumpRequest(...) on the second.
type ContextKey_Formatted_Request = struct{}

func _main() error {
	cfg, err := getConfig()
	if err != nil {
		return err
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
					ContextKey_Formatted_Request{},
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
				request_content := r.Request.Context().Value(ContextKey_Formatted_Request{}).(string)
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
		Addr: cfg.Addr,
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

	log.Printf("cert-file=%s key-file=%s listen-addr=%s", cfg.CertFile, cfg.KeyFile, srv.Addr)
	if err := srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != http.ErrServerClosed {
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
