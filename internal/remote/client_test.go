package remote

import "testing"

func TestNormalizeURL(t *testing.T) {
	if got := NormalizeURL("127.0.0.1:8790"); got != "http://127.0.0.1:8790" {
		t.Fatalf("got=%s", got)
	}
	if got := NormalizeURL("http://127.0.0.1:8790/"); got != "http://127.0.0.1:8790" {
		t.Fatalf("got=%s", got)
	}
}
