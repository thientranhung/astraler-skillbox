package app

import "testing"

func TestPingRegistered(t *testing.T) {
	app := New()
	if !app.HasMethod("ping") {
		t.Fatal("ping must be registered")
	}
}
