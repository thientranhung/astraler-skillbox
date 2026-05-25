package app

import "testing"

func TestPingRegistered(t *testing.T) {
	a := New(nil, nil, nil, nil, nil)
	if !a.HasMethod("ping") {
		t.Fatal("ping must be registered")
	}
}

func TestAllMethodsRegistered(t *testing.T) {
	a := New(nil, nil, nil, nil, nil)
	for _, method := range []string{
		"ping", "host.choose", "host.scan", "skill.list", "settings.get", "operation.cancel",
		"project.add", "project.list", "project.get", "project.scan",
	} {
		if !a.HasMethod(method) {
			t.Errorf("method %q not registered", method)
		}
	}
}
