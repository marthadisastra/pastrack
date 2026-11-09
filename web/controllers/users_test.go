package controllers

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserAndPasswordValidation(t *testing.T) {
	for _, password := range []string{"short", strings.Repeat("a", 73), "12345678901\x00"} {
		if validPassword(password) == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if validPassword("Long passphrase 2026") != nil {
		t.Fatal("valid passphrase rejected")
	}
	u := UserInput{Username: "  TEST.User ", FullName: " Test User ", Email: "TEST@example.com", Role: "viewer"}
	if e := validateUser(&u, true); e != nil || u.Username != "test.user" || u.Email != "test@example.com" {
		t.Fatal(u, e)
	}
	for _, mutate := range []func(*UserInput){func(u *UserInput) { u.Role = "root" }, func(u *UserInput) { u.Email = "not-an-email" }, func(u *UserInput) { u.Username = "x' OR 1=1" }, func(u *UserInput) { u.FullName = "" }} {
		v := u
		mutate(&v)
		if validateUser(&v, true) == nil {
			t.Fatal("invalid user accepted")
		}
	}
}
func TestPasswordAlgorithms(t *testing.T) {
	password := "A secure passphrase 2026"
	hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if e != nil {
		t.Fatal(e)
	}
	if !checkPassword(string(hash), "bcrypt", password) || checkPassword(string(hash), "bcrypt", "wrong") {
		t.Fatal("bcrypt comparison failed")
	}
	if !checkPassword(tokenHash(password), "legacy_sha256", password) || checkPassword(tokenHash(password), "legacy_sha256", "wrong") {
		t.Fatal("legacy migration password comparison failed")
	}
	if checkPassword("bad", "unsupported", password) {
		t.Fatal("unrecognized algorithm accepted")
	}
}
func TestActorIsolation(t *testing.T) {
	for _, name := range []string{"alice", "bob"} {
		r := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(r.Context(), userContextKey{}, &User{ID: 1, Username: name, Active: true, Role: "viewer"})
		if actorName(ctx) != name {
			t.Fatal("audit actor shared between requests")
		}
	}
	if actorName(context.Background()) != "" {
		t.Fatal("missing actor should not become admin")
	}
}
