package netx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-engine/netx/handlers"
	"github.com/ooni/probe-engine/netx/internal/resolver"
	"github.com/ooni/probe-engine/netx/internal/resolver/chainresolver"
	"github.com/ooni/probe-engine/netx/modelx"
)

var (
	dohClientHandle *http.Client
	dohClientOnce   sync.Once
)

func newHTTPClientForDoH(beginning time.Time, handler modelx.Handler) *http.Client {
	if handler == handlers.NoHandler {
		// A bit of extra complexity for a good reason: if the user is not
		// interested into setting a default handler, then it is fine to
		// always return the same *http.Client for DoH. This means that we
		// don't need to care about closing the connections used by this
		// *http.Client, therefore we don't leak resources because we fail
		// to close the idle connections.
		dohClientOnce.Do(func() {
			transport := newHTTPTransport(
				time.Now(),
				handlers.NoHandler,
				newDialer(time.Now(), handler),
				false, // DisableKeepAlives
				http.ProxyFromEnvironment,
			)
			dohClientHandle = &http.Client{Transport: transport}
		})
		return dohClientHandle
	}
	// Otherwise, if the user wants to have a default handler, we
	// return a transport that does not leak connections.
	transport := newHTTPTransport(
		beginning,
		handler,
		newDialer(beginning, handler),
		true, // DisableKeepAlives
		http.ProxyFromEnvironment,
	)
	return &http.Client{Transport: transport}
}

func withPort(address, port string) string {
	// Handle the case where port was not specified. We have written in
	// a bunch of places that we can just pass a domain in this case and
	// so we need to gracefully ensure this is still possible.
	_, _, err := net.SplitHostPort(address)
	if err != nil && strings.Contains(err.Error(), "missing port in address") {
		address = net.JoinHostPort(address, port)
	}
	return address
}

type resolverWrapper struct {
	beginning time.Time
	handler   modelx.Handler
	resolver  modelx.DNSResolver
}

func newResolverWrapper(
	beginning time.Time, handler modelx.Handler,
	resolver modelx.DNSResolver,
) *resolverWrapper {
	return &resolverWrapper{
		beginning: beginning,
		handler:   handler,
		resolver:  resolver,
	}
}

// LookupAddr returns the name of the provided IP address
func (r *resolverWrapper) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a host
func (r *resolverWrapper) LookupCNAME(ctx context.Context, host string) (string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupCNAME(ctx, host)
}

// LookupHost returns the IP addresses of a host
func (r *resolverWrapper) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupHost(ctx, hostname)
}

// LookupMX returns the MX records of a specific name
func (r *resolverWrapper) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (r *resolverWrapper) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupNS(ctx, name)
}

func newResolver(
	beginning time.Time, handler modelx.Handler, network, address string,
) (modelx.DNSResolver, error) {
	// Implementation note: system need to be dealt with
	// separately because it doesn't have any transport.
	if network == "system" || network == "" {
		return newResolverWrapper(
			beginning, handler, resolver.NewResolverSystem()), nil
	}
	if network == "doh" {
		return newResolverWrapper(beginning, handler, resolver.NewResolverHTTPS(
			newHTTPClientForDoH(beginning, handler), address,
		)), nil
	}
	if network == "dot" {
		// We need a child dialer here to avoid an endless loop where the
		// dialer will ask us to resolve, we'll tell the dialer to dial, it
		// will ask us to resolve, ...
		return newResolverWrapper(beginning, handler, resolver.NewResolverTLS(
			newDialer(beginning, handler), withPort(address, "853"),
		)), nil
	}
	if network == "tcp" {
		// Same rationale as above: avoid possible endless loop
		return newResolverWrapper(beginning, handler, resolver.NewResolverTCP(
			newDialer(beginning, handler), withPort(address, "53"),
		)), nil
	}
	if network == "udp" {
		// Same rationale as above: avoid possible endless loop
		return newResolverWrapper(beginning, handler, resolver.NewResolverUDP(
			newDialer(beginning, handler), withPort(address, "53"),
		)), nil
	}
	return nil, errors.New("resolver.New: unsupported network value")
}

// NewResolver creates a standalone Resolver
func NewResolver(network, address string) (modelx.DNSResolver, error) {
	return newResolver(time.Now(), handlers.NoHandler, network, address)
}

// ChainResolvers chains a primary and a secondary resolver such that
// we can fallback to the secondary if primary is broken.
func ChainResolvers(primary, secondary modelx.DNSResolver) modelx.DNSResolver {
	return chainresolver.New(primary, secondary)
}
