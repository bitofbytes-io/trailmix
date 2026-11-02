package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeFailsFastWhenPortInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	httpServer := &http.Server{Addr: listener.Addr().String(), Handler: http.NotFoundHandler()}
	done := make(chan error, 1)
	go func() { done <- serve(context.Background(), httpServer) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("serve returned nil, want a listen error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the listen failure")
	}
}
