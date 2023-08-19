package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

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
