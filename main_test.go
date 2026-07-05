package main

import "testing"

func TestServiceName(t *testing.T) {
	if serviceName == "" {
		t.Fatal("serviceName should not be empty")
	}
}
