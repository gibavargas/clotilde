package admin

import (
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "s3cret")
	SetDefaultHandler(nil) // rebuild singleton from env

	if !ValidateCredentials("admin", "s3cret") {
		t.Fatal("valid credentials should pass")
	}
	if ValidateCredentials("admin", "wrong") {
		t.Fatal("wrong password must fail")
	}
	if ValidateCredentials("nope", "s3cret") {
		t.Fatal("wrong username must fail")
	}
	if ValidateCredentials("", "") {
		t.Fatal("empty credentials must fail")
	}

	// Unconfigured admin must fail closed.
	t.Setenv("ADMIN_USER", "")
	t.Setenv("ADMIN_PASSWORD", "")
	SetDefaultHandler(nil)
	if ValidateCredentials("admin", "s3cret") {
		t.Fatal("unconfigured admin must fail closed")
	}
}
