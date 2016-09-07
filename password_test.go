package main

import "testing"

func TestPassword(t *testing.T) {
	p1 := newHashPassword("test")
	if !p1.Equal("test") {
		t.Fatal("Bad password compare function")
	}
	if p1.Equal("test1") {
		t.Fatal("Bad password compare function")
	}
}
