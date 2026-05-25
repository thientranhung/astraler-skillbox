package providers

// InstallTarget is read-only core metadata for future install flows.
// It is not persisted, not exposed through JSON-RPC in Slice 2E, and must not replace provider keys.
type InstallTarget struct {
	ID                 string
	ProviderKey        string
	DisplayName        string
	RelativeSkillsPath string
	CompatibleLabels   []string
}

func InstallTargets() []InstallTarget {
	return []InstallTarget{
		{
			ID:                 "shared_agents",
			ProviderKey:        GenericAgentsKey,
			DisplayName:        "Shared Agent Skills (.agents)",
			RelativeSkillsPath: GenericAgentsSkillsPath,
			CompatibleLabels:   []string{"Codex", "Antigravity", "compatible agents"},
		},
		{
			ID:                 ClaudeKey,
			ProviderKey:        ClaudeKey,
			DisplayName:        "Claude (.claude)",
			RelativeSkillsPath: ClaudeSkillsPath,
			CompatibleLabels:   []string{"Claude"},
		},
	}
}

func InstallTargetByProviderKey(providerKey string) (InstallTarget, bool) {
	for _, target := range InstallTargets() {
		if target.ProviderKey == providerKey {
			return target, true
		}
	}
	return InstallTarget{}, false
}
