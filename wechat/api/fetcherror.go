// Package api provides WeChat API client and types.
package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// FetchErrorType categorizes a network-layer (non-HTTP-status) fetch failure.
//
// Mirrors openclaw-weixin's classifyFetchError (v2.4.5), but implemented with
// Go's standard error types (net.DNSError, net.OpError) rather than errno
// string matching.
type FetchErrorType string

const (
	// FetchErrDNS: DNS resolution failed (e.g. ENOTFOUND, EAI_AGAIN).
	FetchErrDNS FetchErrorType = "dns"
	// FetchErrTCP: TCP connection refused, timed out, or unreachable.
	FetchErrTCP FetchErrorType = "tcp"
	// FetchErrTLS: TLS handshake / certificate failure.
	FetchErrTLS FetchErrorType = "tls"
	// FetchErrTimeout: request deadline exceeded (incl. context cancellation).
	FetchErrTimeout FetchErrorType = "timeout"
	// FetchErrUnknown: unclassified network error.
	FetchErrUnknown FetchErrorType = "unknown"
)

// FetchError wraps a network-layer fetch failure with a structured category,
// a short human description, and a best-effort underlying code.
type FetchError struct {
	Type        FetchErrorType
	Description string
	Code        string // underlying code string, best-effort (may be empty)
	Err         error
}

// Error implements the error interface.
func (e *FetchError) Error() string {
	if e == nil || e.Err == nil {
		return string(e.Type)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Err.Error())
}

// Unwrap exposes the underlying error for errors.Is/As.
func (e *FetchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ClassifyFetchError inspects a fetch/network-level error and returns a typed
// FetchError. It looks at:
//   - context deadline / cancellation  -> timeout
//   - *net.DNSError                     -> dns
//   - *net.OpError with a TLS-ish cause -> tls
//   - connection refused / unreachable  -> tcp
//   - otherwise                          -> unknown
//
// err may be nil; ClassifyFetchError(nil) returns nil.
func ClassifyFetchError(err error) *FetchError {
	if err == nil {
		return nil
	}

	// Timeout / cancellation (covers both our own context deadlines and
	// external abort via context cancellation).
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &FetchError{
			Type:        FetchErrTimeout,
			Description: "request timeout",
			Err:         err,
		}
	}

	// DNS resolution failure.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &FetchError{
			Type:        FetchErrDNS,
			Description: "dns resolution failed",
			Code:        dnsErr.Err,
			Err:         err,
		}
	}

	// net.OpError covers most connection-layer failures; classify by inspecting
	// the nested cause for TLS markers vs plain TCP failures.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if isTLSError(opErr.Err) {
			return &FetchError{
				Type:        FetchErrTLS,
				Description: "tls handshake failed",
				Code:        opErrorString(opErr),
				Err:         err,
			}
		}
		return &FetchError{
			Type:        FetchErrTCP,
			Description: "tcp connection failed",
			Code:        opErrorString(opErr),
			Err:         err,
		}
	}

	// Fallback: inspect the error string for TLS / connection markers, matching
	// the upstream behavior for opaque errors from the HTTP client / TLS stack.
	msg := err.Error()
	if isTLSString(msg) {
		return &FetchError{Type: FetchErrTLS, Description: "tls error", Err: err}
	}
	if isTCPString(msg) {
		return &FetchError{Type: FetchErrTCP, Description: "tcp error", Err: err}
	}

	return &FetchError{
		Type:        FetchErrUnknown,
		Description: "unknown network error",
		Err:         err,
	}
}

// isTLSError reports whether the nested cause looks like a TLS failure.
func isTLSError(cause error) bool {
	if cause == nil {
		return false
	}
	// x509 errors implement the error interface with recognizable messages.
	return isTLSString(cause.Error())
}

// isTLSString matches TLS / certificate markers from the upstream classifier.
func isTLSString(s string) bool {
	up := strings.ToUpper(s)
	for _, marker := range []string{"SSL", "TLS", "CERT", "X509", "UNABLE_TO_VERIFY", "DEPTH_ZERO"} {
		if strings.Contains(up, marker) {
			return true
		}
	}
	return false
}

// isTCPString matches TCP connection markers (refused / unreachable / reset).
func isTCPString(s string) bool {
	up := strings.ToUpper(s)
	for _, marker := range []string{"REFUSED", "UNREACHABLE", "RESET", "BROKEN PIPE", "NO SUCH HOST"} {
		if strings.Contains(up, marker) {
			return true
		}
	}
	return false
}

// opErrorString returns a best-effort code string from a net.OpError,
// preferring the nested cause's text over the (often empty) syscall code.
func opErrorString(opErr *net.OpError) string {
	if opErr == nil {
		return ""
	}
	if opErr.Err != nil {
		return opErr.Err.Error()
	}
	return opErr.Op
}
