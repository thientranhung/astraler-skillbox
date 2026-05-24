package handlers

import "testing"

func TestPingReturnsPong(t *testing.T) {
	got := Ping()
	if !got.Pong {
		t.Fatal("expected pong")
	}
	if got.TS == "" {
		t.Fatal("expected timestamp")
	}
}
