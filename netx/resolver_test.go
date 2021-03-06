package netx_test

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/probe-engine/netx"
	"github.com/ooni/probe-engine/netx/handlers"
	"github.com/ooni/probe-engine/netx/internal/resolver/brokenresolver"
)

func TestIntegrationResolverLookupAddr(t *testing.T) {
	resolver, err := netx.NewResolver("system", "")
	if err != nil {
		t.Fatal(err)
	}
	names, err := resolver.LookupAddr(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 1 {
		t.Fatal("unexpected result")
	}
}

func TestIntegrationResolverLookupCNAME(t *testing.T) {
	resolver, err := netx.NewResolver("system", "")
	if err != nil {
		t.Fatal(err)
	}
	cname, err := resolver.LookupCNAME(context.Background(), "www.ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if cname == "" {
		t.Fatal("unexpected result")
	}
}

func testresolverquick(t *testing.T, network, address string) {
	resolver, err := netx.NewResolver(network, address)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}
	addrs, err := resolver.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil addrs here")
	}
	var foundquad8 bool
	for _, addr := range addrs {
		if addr == "8.8.8.8" {
			foundquad8 = true
		}
	}
	if !foundquad8 {
		t.Fatal("did not find 8.8.8.8 in ouput")
	}
}

func TestIntegrationNewResolverUDPAddress(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8:53")
}

func TestIntegrationNewResolverUDPAddressNoPort(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8")
}

func TestIntegrationNewResolverUDPDomain(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com:53")
}

func TestIntegrationNewResolverUDPDomainNoPort(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com")
}

func TestIntegrationNewResolverSystem(t *testing.T) {
	testresolverquick(t, "system", "")
}

func TestIntegrationNewResolverTCPAddress(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8:53")
}

func TestIntegrationNewResolverTCPAddressNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8")
}

func TestIntegrationNewResolverTCPDomain(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com:53")
}

func TestIntegrationNewResolverTCPDomainNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com")
}

func TestIntegrationNewResolverDoTAddress(t *testing.T) {
	testresolverquick(t, "dot", "9.9.9.9:853")
}

func TestIntegrationNewResolverDoTAddressNoPort(t *testing.T) {
	testresolverquick(t, "dot", "9.9.9.9")
}

func TestIntegrationNewResolverDoTDomain(t *testing.T) {
	testresolverquick(t, "dot", "dns.quad9.net:853")
}

func TestIntegrationNewResolverDoTDomainNoPort(t *testing.T) {
	testresolverquick(t, "dot", "dns.quad9.net")
}

func TestIntegrationNewResolverDoH(t *testing.T) {
	testresolverquick(t, "doh", "https://cloudflare-dns.com/dns-query")
}

func TestIntegrationNewResolverInvalid(t *testing.T) {
	resolver, err := netx.NewResolver(
		"antani", "https://cloudflare-dns.com/dns-query",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}

func TestIntegrationChainResolvers(t *testing.T) {
	fallback, err := netx.NewResolver("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	primary := brokenresolver.New()
	dialer := netx.NewDialer()
	resolver := netx.ChainResolvers(primary, fallback)
	dialer.SetResolver(resolver)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err) // we don't expect error because good resolver is first
	}
	if primary.NumErrors.Load() < 1 {
		t.Fatal("primary has not been used")
	}
	defer conn.Close()
}

func TestIntegrationResolverLookupMX(t *testing.T) {
	resolver, err := netx.NewResolver("system", "")
	if err != nil {
		t.Fatal(err)
	}
	records, err := resolver.LookupMX(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 1 {
		t.Fatal("unexpected result")
	}
}

func TestIntegrationResolverLookupNS(t *testing.T) {
	resolver, err := netx.NewResolver("system", "")
	if err != nil {
		t.Fatal(err)
	}
	records, err := resolver.LookupNS(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 1 {
		t.Fatal("unexpected result")
	}
}

func TestUnitNewHTTPClientForDoH(t *testing.T) {
	first := netx.NewHTTPClientForDoH(
		time.Now(), handlers.NoHandler,
	)
	second := netx.NewHTTPClientForDoH(
		time.Now(), handlers.NoHandler,
	)
	if first != second {
		t.Fatal("expected to see same client here")
	}
	third := netx.NewHTTPClientForDoH(
		time.Now(), handlers.StdoutHandler,
	)
	if first == third {
		t.Fatal("expected to see different client here")
	}
}
