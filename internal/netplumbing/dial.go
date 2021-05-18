package netplumbing

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrProxyNotImplemented indicates that we don't support connecting via proxy.
var ErrProxyNotImplemented = errors.New("netplumbing: proxy not implemented")

// ErrDial is an error occurred when dialing.
type ErrDial struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrDial) Unwrap() error {
	return err.error
}

// DialContext dials a cleartext connection.
func (txp *Transport) DialContext(
	ctx context.Context, network string, address string) (net.Conn, error) {
	if settings := ContextSettings(ctx); settings != nil && settings.Proxy != nil {
		return nil, &ErrDial{ErrProxyNotImplemented}
	}
	conn, err := txp.directDialContext(ctx, network, address)
	if err != nil {
		return nil, &ErrDial{err}
	}
	return conn, nil
}

// directDialContext is a dial context that does not use a proxy.
func (txp *Transport) directDialContext(
	ctx context.Context, network string, address string) (net.Conn, error) {
	log := txp.logger(ctx)
	log.Debugf("dial: %s/%s...", address, network)
	conn, err := txp.doDialContext(ctx, network, address)
	if err != nil {
		log.Debugf("dial: %s/%s... %s", address, network, err)
		return nil, err
	}
	log.Debugf("dial: %s/%s... ok", address, network)
	return conn, nil
}

// ErrAllConnectsFailed indicates that all connects failed.
type ErrAllConnectsFailed struct {
	// Errors contains all the errors that occurred.
	Errors []error
}

// Error implements error.Error.
func (err *ErrAllConnectsFailed) Error() string {
	return fmt.Sprintf("one or more connect() failed: %#v", err.Errors)
}

// doDialContext implements dialContext.
func (txp *Transport) doDialContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ipaddrs, err := txp.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	aggregate := &ErrAllConnectsFailed{}
	for _, ipaddr := range ipaddrs {
		epnt := net.JoinHostPort(ipaddr, port)
		conn, err := txp.connect(ctx, network, epnt)
		if err == nil {
			return conn, nil
		}
		aggregate.Errors = append(aggregate.Errors, err)
	}
	return nil, aggregate
}
