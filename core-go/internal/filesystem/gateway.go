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

// ValidateProjectPath delegates to the package-level function.
// Unlike ValidateHostPath it does NOT require the path to be writable.
func (g *Gateway) ValidateProjectPath(path string) error {
	return ValidateProjectPath(path)
}

// PathInfo delegates to StatPathInfo.
func (g *Gateway) PathInfo(path string) (PathInfo, error) {
	return StatPathInfo(path)
}

// ListSkillEntries delegates to ScanProjectSkills.
func (g *Gateway) ListSkillEntries(skillsPath string) ([]ProjectEntry, error) {
	return ScanProjectSkills(skillsPath)
}

// LstatExists delegates to the package-level function.
func (g *Gateway) LstatExists(path string) (bool, error) {
	return LstatExists(path)
}

// EnsureDir delegates to the package-level function.
func (g *Gateway) EnsureDir(path string) error {
	return EnsureDir(path)
}

// CreateSymlink delegates to the package-level function.
func (g *Gateway) CreateSymlink(source, linkPath string) error {
	return CreateSymlink(source, linkPath)
}

// ResolveEntry delegates to the package-level function.
func (g *Gateway) ResolveEntry(path string) (EntryFacts, error) {
	return ResolveEntry(path)
}

// RemoveSymlink delegates to the package-level function.
func (g *Gateway) RemoveSymlink(path string) error {
	return RemoveSymlink(path)
}
