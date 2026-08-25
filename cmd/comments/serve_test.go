package main

import "testing"

func TestValidateLoopbackAddr(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", "localhost:8080", "[::1]:9000"} {
		if err := validateLoopbackAddr(addr); err != nil {
			t.Errorf("validateLoopbackAddr(%q): %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:8080", "192.168.1.2:8080", "example.com:80", "bad"} {
		if err := validateLoopbackAddr(addr); err == nil {
			t.Errorf("validateLoopbackAddr(%q) unexpectedly passed", addr)
		}
	}
}
