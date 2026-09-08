package core

import "testing"

func TestLoopbackToDockerHost(t *testing.T) {
	got := loopbackToDockerHost("http://127.0.0.1:8790")
	if got != "http://host.docker.internal:8790" {
		t.Fatalf("got %s", got)
	}
	if loopbackToDockerHost("http://box:8790") != "" {
		t.Fatal("non-loopback should be empty")
	}
}
