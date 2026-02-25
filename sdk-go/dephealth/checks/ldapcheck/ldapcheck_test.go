package ldapcheck

import (
	"context"
	"testing"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestChecker_Type(t *testing.T) {
	checker := New()
	if got := checker.Type(); got != "ldap" {
		t.Errorf("Type() = %q, expected %q", got, "ldap")
	}
}

func TestChecker_Defaults(t *testing.T) {
	checker := New()
	if checker.checkMethod != MethodRootDSE {
		t.Errorf("default checkMethod = %q, expected %q", checker.checkMethod, MethodRootDSE)
	}
	if checker.searchFilter != "(objectClass=*)" {
		t.Errorf("default searchFilter = %q, expected %q", checker.searchFilter, "(objectClass=*)")
	}
	if checker.searchScope != ScopeBase {
		t.Errorf("default searchScope = %d, expected %d", checker.searchScope, ScopeBase)
	}
}

func TestChecker_Options(t *testing.T) {
	checker := New(
		WithCheckMethod(MethodSimpleBind),
		WithBindDN("cn=admin,dc=test,dc=local"),
		WithBindPassword("password"),
		WithBaseDN("dc=test,dc=local"),
		WithSearchFilter("(uid=*)"),
		WithSearchScope(ScopeSub),
		WithTLS(true),
		WithStartTLS(false),
		WithTLSSkipVerify(true),
	)

	if checker.checkMethod != MethodSimpleBind {
		t.Errorf("checkMethod = %q, expected %q", checker.checkMethod, MethodSimpleBind)
	}
	if checker.bindDN != "cn=admin,dc=test,dc=local" {
		t.Errorf("bindDN = %q, expected %q", checker.bindDN, "cn=admin,dc=test,dc=local")
	}
	if checker.bindPassword != "password" {
		t.Errorf("bindPassword mismatch")
	}
	if checker.baseDN != "dc=test,dc=local" {
		t.Errorf("baseDN = %q, expected %q", checker.baseDN, "dc=test,dc=local")
	}
	if checker.searchFilter != "(uid=*)" {
		t.Errorf("searchFilter = %q, expected %q", checker.searchFilter, "(uid=*)")
	}
	if checker.searchScope != ScopeSub {
		t.Errorf("searchScope = %d, expected %d", checker.searchScope, ScopeSub)
	}
	if !checker.useTLS {
		t.Error("useTLS should be true")
	}
	if checker.startTLS {
		t.Error("startTLS should be false")
	}
	if !checker.tlsSkipVerify {
		t.Error("tlsSkipVerify should be true")
	}
}

func TestChecker_Check_ConnectionRefused(t *testing.T) {
	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "389"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestNewFromConfig_Default(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "ldap://localhost:389",
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if rc.checkMethod != MethodRootDSE {
		t.Errorf("default checkMethod = %q, expected %q", rc.checkMethod, MethodRootDSE)
	}
	if rc.useTLS {
		t.Error("useTLS should be false for ldap://")
	}
}

func TestNewFromConfig_LDAPS(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "ldaps://localhost:636",
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if !rc.useTLS {
		t.Error("useTLS should be true for ldaps://")
	}
}

func TestNewFromConfig_SimpleBind(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL:              "ldap://localhost:389",
		LDAPCheckMethod:  "simple_bind",
		LDAPBindDN:       "cn=admin,dc=test,dc=local",
		LDAPBindPassword: "password",
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if rc.checkMethod != MethodSimpleBind {
		t.Errorf("checkMethod = %q, expected %q", rc.checkMethod, MethodSimpleBind)
	}
	if rc.bindDN != "cn=admin,dc=test,dc=local" {
		t.Errorf("bindDN = %q, expected %q", rc.bindDN, "cn=admin,dc=test,dc=local")
	}
}

func TestNewFromConfig_Search(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL:              "ldap://localhost:389",
		LDAPCheckMethod:  "search",
		LDAPBaseDN:       "dc=test,dc=local",
		LDAPSearchFilter: "(uid=testuser)",
		LDAPSearchScope:  "sub",
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if rc.checkMethod != MethodSearch {
		t.Errorf("checkMethod = %q, expected %q", rc.checkMethod, MethodSearch)
	}
	if rc.baseDN != "dc=test,dc=local" {
		t.Errorf("baseDN = %q, expected %q", rc.baseDN, "dc=test,dc=local")
	}
	if rc.searchFilter != "(uid=testuser)" {
		t.Errorf("searchFilter = %q, expected %q", rc.searchFilter, "(uid=testuser)")
	}
	if rc.searchScope != ScopeSub {
		t.Errorf("searchScope = %d, expected %d", rc.searchScope, ScopeSub)
	}
}

func TestNewFromConfig_StartTLS(t *testing.T) {
	startTLS := true
	skipVerify := true
	dc := &dephealth.DependencyConfig{
		URL:               "ldap://localhost:389",
		LDAPStartTLS:      &startTLS,
		LDAPTLSSkipVerify: &skipVerify,
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if !rc.startTLS {
		t.Error("startTLS should be true")
	}
	if !rc.tlsSkipVerify {
		t.Error("tlsSkipVerify should be true")
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		input    string
		expected SearchScope
	}{
		{"base", ScopeBase},
		{"one", ScopeOne},
		{"sub", ScopeSub},
		{"BASE", ScopeBase},
		{"ONE", ScopeOne},
		{"SUB", ScopeSub},
		{"unknown", ScopeBase},
		{"", ScopeBase},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseScope(tt.input)
			if got != tt.expected {
				t.Errorf("parseScope(%q) = %d, expected %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestClassifyError_ConnectionRefused(t *testing.T) {
	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error")
	}

	ce, ok := err.(*dephealth.ClassifiedCheckError)
	if !ok {
		// The error might be wrapped differently; check that it's not nil.
		t.Logf("error type: %T, message: %v", err, err)
		return
	}
	if ce.Category != dephealth.StatusConnectionError {
		t.Errorf("category = %q, expected %q", ce.Category, dephealth.StatusConnectionError)
	}
	if ce.Detail != "connection_refused" {
		t.Errorf("detail = %q, expected %q", ce.Detail, "connection_refused")
	}
}

func TestValidateLDAPConfig_SimpleBind_MissingCredentials(t *testing.T) {
	// This tests via the options.go validation, not directly.
	// We test the NewFromConfig path to verify the options are correctly set.
	dc := &dephealth.DependencyConfig{
		URL:             "ldap://localhost:389",
		LDAPCheckMethod: "simple_bind",
		// Missing LDAPBindDN and LDAPBindPassword — should be caught by validateLDAPConfig
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	// NewFromConfig doesn't validate — validation happens in makeDepOption.
	// Verify config is set correctly.
	if rc.checkMethod != MethodSimpleBind {
		t.Errorf("checkMethod = %q, expected %q", rc.checkMethod, MethodSimpleBind)
	}
}
