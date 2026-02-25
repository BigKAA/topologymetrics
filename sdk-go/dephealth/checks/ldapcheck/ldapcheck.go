// Package ldapcheck provides an LDAP health checker for dephealth.
//
// Supports four check methods: anonymous_bind, simple_bind, root_dse, search.
// Supports LDAP (plain), LDAPS (TLS), and StartTLS connections.
//
// Import this package to register the LDAP checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"
package ldapcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeLDAP, NewFromConfig)
}

// CheckMethod represents the LDAP check method.
type CheckMethod string

const (
	MethodAnonymousBind CheckMethod = "anonymous_bind"
	MethodSimpleBind    CheckMethod = "simple_bind"
	MethodRootDSE       CheckMethod = "root_dse"
	MethodSearch        CheckMethod = "search"
)

// SearchScope represents the LDAP search scope.
type SearchScope int

const (
	ScopeBase SearchScope = ldap.ScopeBaseObject
	ScopeOne  SearchScope = ldap.ScopeSingleLevel
	ScopeSub  SearchScope = ldap.ScopeWholeSubtree
)

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks against an LDAP server.
// Supports two modes:
//   - Standalone: creates a new LDAP connection per check
//   - Pool: uses an existing *ldap.Conn
type Checker struct {
	conn          *ldap.Conn  // nil = standalone, non-nil = pool mode
	checkMethod   CheckMethod // default: root_dse
	bindDN        string
	bindPassword  string
	baseDN        string
	searchFilter  string // default: (objectClass=*)
	searchScope   SearchScope
	useTLS        bool // true for ldaps:// scheme
	startTLS      bool // true for StartTLS
	tlsSkipVerify bool
}

// WithConn sets an existing LDAP connection for pool mode.
func WithConn(conn *ldap.Conn) Option {
	return func(c *Checker) {
		c.conn = conn
	}
}

// WithCheckMethod sets the LDAP check method.
func WithCheckMethod(method CheckMethod) Option {
	return func(c *Checker) {
		c.checkMethod = method
	}
}

// WithBindDN sets the DN for Simple Bind.
func WithBindDN(dn string) Option {
	return func(c *Checker) {
		c.bindDN = dn
	}
}

// WithBindPassword sets the password for Simple Bind.
func WithBindPassword(password string) Option {
	return func(c *Checker) {
		c.bindPassword = password
	}
}

// WithBaseDN sets the base DN for search method.
func WithBaseDN(baseDN string) Option {
	return func(c *Checker) {
		c.baseDN = baseDN
	}
}

// WithSearchFilter sets the LDAP search filter.
func WithSearchFilter(filter string) Option {
	return func(c *Checker) {
		c.searchFilter = filter
	}
}

// WithSearchScope sets the LDAP search scope.
func WithSearchScope(scope SearchScope) Option {
	return func(c *Checker) {
		c.searchScope = scope
	}
}

// WithTLS enables TLS (for ldaps:// connections).
func WithTLS(enabled bool) Option {
	return func(c *Checker) {
		c.useTLS = enabled
	}
}

// WithStartTLS enables StartTLS (only with ldap://).
func WithStartTLS(enabled bool) Option {
	return func(c *Checker) {
		c.startTLS = enabled
	}
}

// WithTLSSkipVerify disables TLS certificate verification.
func WithTLSSkipVerify(skip bool) Option {
	return func(c *Checker) {
		c.tlsSkipVerify = skip
	}
}

// New creates a new LDAP health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{
		checkMethod:  MethodRootDSE,
		searchFilter: "(objectClass=*)",
		searchScope:  ScopeBase,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates an LDAP checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option

	if dc.LDAPCheckMethod != "" {
		opts = append(opts, WithCheckMethod(CheckMethod(dc.LDAPCheckMethod)))
	}
	if dc.LDAPBindDN != "" {
		opts = append(opts, WithBindDN(dc.LDAPBindDN))
	}
	if dc.LDAPBindPassword != "" {
		opts = append(opts, WithBindPassword(dc.LDAPBindPassword))
	}
	if dc.LDAPBaseDN != "" {
		opts = append(opts, WithBaseDN(dc.LDAPBaseDN))
	}
	if dc.LDAPSearchFilter != "" {
		opts = append(opts, WithSearchFilter(dc.LDAPSearchFilter))
	}
	if dc.LDAPSearchScope != "" {
		opts = append(opts, WithSearchScope(parseScope(dc.LDAPSearchScope)))
	}
	if dc.LDAPStartTLS != nil && *dc.LDAPStartTLS {
		opts = append(opts, WithStartTLS(true))
	}
	if dc.LDAPTLSSkipVerify != nil && *dc.LDAPTLSSkipVerify {
		opts = append(opts, WithTLSSkipVerify(true))
	}

	// Detect ldaps:// scheme from URL.
	if dc.URL != "" && strings.HasPrefix(strings.ToLower(dc.URL), "ldaps://") {
		opts = append(opts, WithTLS(true))
	}

	return New(opts...)
}

// Check performs an LDAP health check against the given endpoint.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.conn != nil {
		return c.checkWithConn(c.conn)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *Checker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)

	conn, err := c.dial(ctx, addr)
	if err != nil {
		return classifyError(err, addr)
	}
	defer func() { _ = conn.Close() }()

	if c.startTLS {
		tlsCfg := &tls.Config{
			ServerName:         endpoint.Host,
			InsecureSkipVerify: c.tlsSkipVerify, //nolint:gosec // user-configurable
		}
		if err := conn.StartTLS(tlsCfg); err != nil {
			return classifyError(err, addr)
		}
	}

	if err := c.checkWithConn(conn); err != nil {
		return classifyError(err, addr)
	}
	return nil
}

func (c *Checker) dial(ctx context.Context, addr string) (*ldap.Conn, error) {
	// Use a dial timeout shorter than the context timeout for classifiable errors.
	dialTimeout := 3 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < dialTimeout {
			dialTimeout = remaining
		}
	}

	dialer := &net.Dialer{Timeout: dialTimeout}

	if c.useTLS {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.tlsSkipVerify, //nolint:gosec // user-configurable
		}
		return ldap.DialURL("ldaps://"+addr,
			ldap.DialWithDialer(dialer),
			ldap.DialWithTLSConfig(tlsCfg),
		)
	}

	return ldap.DialURL("ldap://"+addr,
		ldap.DialWithDialer(dialer),
	)
}

func (c *Checker) checkWithConn(conn *ldap.Conn) error {
	switch c.checkMethod {
	case MethodAnonymousBind:
		return conn.UnauthenticatedBind("")
	case MethodSimpleBind:
		return conn.Bind(c.bindDN, c.bindPassword)
	case MethodRootDSE:
		return c.searchRootDSE(conn)
	case MethodSearch:
		return c.searchWithConfig(conn)
	default:
		return c.searchRootDSE(conn)
	}
}

func (c *Checker) searchRootDSE(conn *ldap.Conn) error {
	req := ldap.NewSearchRequest(
		"",
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, // sizeLimit
		0, // timeLimit
		false,
		"(objectClass=*)",
		[]string{"namingContexts", "subschemaSubentry"},
		nil,
	)
	_, err := conn.Search(req)
	return err
}

func (c *Checker) searchWithConfig(conn *ldap.Conn) error {
	// Bind before search if credentials are provided.
	if c.bindDN != "" {
		if err := conn.Bind(c.bindDN, c.bindPassword); err != nil {
			return err
		}
	}

	filter := c.searchFilter
	if filter == "" {
		filter = "(objectClass=*)"
	}

	req := ldap.NewSearchRequest(
		c.baseDN,
		int(c.searchScope),
		ldap.NeverDerefAliases,
		1, // sizeLimit
		0, // timeLimit
		false,
		filter,
		[]string{"dn"},
		nil,
	)
	_, err := conn.Search(req)
	return err
}

// classifyError wraps LDAP errors with appropriate classification.
func classifyError(err error, target string) error {
	if err == nil {
		return nil
	}

	// LDAP result code errors.
	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) {
		switch ldapErr.ResultCode {
		case ldap.LDAPResultInvalidCredentials: // 49
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusAuthError,
				Detail:   "auth_error",
				Cause:    fmt.Errorf("ldap %s: %w", target, err),
			}
		case ldap.LDAPResultInsufficientAccessRights: // 50
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusAuthError,
				Detail:   "auth_error",
				Cause:    fmt.Errorf("ldap %s: %w", target, err),
			}
		case ldap.LDAPResultBusy, ldap.LDAPResultUnavailable, ldap.LDAPResultUnwillingToPerform:
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusUnhealthy,
				Detail:   "unhealthy",
				Cause:    fmt.Errorf("ldap %s: %w", target, err),
			}
		}
	}

	// Connection refused.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusConnectionError,
				Detail:   "connection_refused",
				Cause:    fmt.Errorf("ldap %s: %w", target, err),
			}
		}
		if opErr.Timeout() {
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusConnectionError,
				Detail:   "connection_refused",
				Cause:    fmt.Errorf("ldap %s: %w", target, err),
			}
		}
	}

	// DNS errors.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusDNSError,
			Detail:   "dns_error",
			Cause:    fmt.Errorf("ldap %s: %w", target, err),
		}
	}

	// TLS errors.
	msg := err.Error()
	if strings.Contains(msg, "tls:") || strings.Contains(msg, "x509:") || strings.Contains(msg, "certificate") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusTLSError,
			Detail:   "tls_error",
			Cause:    fmt.Errorf("ldap %s: %w", target, err),
		}
	}

	// Message-based fallback for connection refused.
	if strings.Contains(msg, "connection refused") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusConnectionError,
			Detail:   "connection_refused",
			Cause:    fmt.Errorf("ldap %s: %w", target, err),
		}
	}

	// Context deadline exceeded.
	if errors.Is(err, context.DeadlineExceeded) {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusConnectionError,
			Detail:   "connection_refused",
			Cause:    fmt.Errorf("ldap %s: %w", target, err),
		}
	}

	return fmt.Errorf("ldap %s: %w", target, err)
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeLDAP)
}

// parseScope converts a scope string to SearchScope.
func parseScope(s string) SearchScope {
	switch strings.ToLower(s) {
	case "one":
		return ScopeOne
	case "sub":
		return ScopeSub
	default:
		return ScopeBase
	}
}
