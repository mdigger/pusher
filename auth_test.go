package main

import (
	"fmt"
	"testing"
)

func TestAuth(t *testing.T) {
	var auth Authorization
	if !auth.SetAdmin("login", "password") {
		t.Error("Set admin failed")
	}
	if !auth.IsAdminRequired() {
		t.Error("Set admin failed")
	}
	if !auth.AuthorizeAdmin("login", "password") {
		t.Error("Authorize admin failed")
	}
	if !auth.AddUser("login", "password") {
		t.Error("Add user failed")
	}
	if !auth.AddUser("login2", "password2") {
		t.Error("Add user 2 failed")
	}
	if auth.AddUser("login2", "password") {
		t.Error("Change user password failed")
	}
	if !auth.RemoveUser("login2") {
		t.Error("Remove user failed")
	}
	if !auth.Authorize("login", "password") {
		t.Error("Authorize user failed")
	}
	fmt.Println(auth.Users())
}
