package app

import "testing"

func TestPingRegistered(t *testing.T) {
	a := New(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if !a.HasMethod("ping") {
		t.Fatal("ping must be registered")
	}
}

func TestAllMethodsRegistered(t *testing.T) {
	a := New(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	for _, method := range []string{
		"ping", "host.choose", "host.scan", "skill.list", "skill.get", "settings.get", "operation.cancel",
		"project.add", "project.list", "project.get", "project.scan", "project.remove",
		"install.skill",
		"remove.skill",
		"dashboard.get",
		"global.scan",
		"global.list",
		"provider.list",
		"providerPlugin.scanGlobal",
		"providerPlugin.list",
		"providerPlugin.setEnabled",
		"providerPlugin.removeOverride",
	} {
		if !a.HasMethod(method) {
			t.Errorf("method %q not registered", method)
		}
	}
}
