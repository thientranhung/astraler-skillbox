package filesystem

// Gateway provides all filesystem operations used by services.
// It is the single point through which services touch the filesystem.
type Gateway struct{}

// NewGateway creates a new Gateway.
func NewGateway() *Gateway {
	return &Gateway{}
}

// ValidateHostPath delegates to the package-level function.
func (g *Gateway) ValidateHostPath(path string) error {
	return ValidateHostPath(path)
}

// EnsureAgentsSkills delegates to the package-level function.
func (g *Gateway) EnsureAgentsSkills(hostPath string) (bool, error) {
	return EnsureAgentsSkills(hostPath)
}

// ScanHostFolder delegates to the package-level function.
func (g *Gateway) ScanHostFolder(skillsPath string) ([]HostEntry, error) {
	return ScanHostFolder(skillsPath)
}

// NormalizeAbs delegates to the package-level function.
func (g *Gateway) NormalizeAbs(path string) (string, error) {
	return NormalizeAbs(path)
}
