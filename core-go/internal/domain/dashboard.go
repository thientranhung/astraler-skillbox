package domain

// InstallModeCounts holds counts of skill installs grouped by delivery mode.
type InstallModeCounts struct {
	Symlink   int
	RsyncCopy int
	Direct    int
}

// WarningSeverityCounts holds counts of active warnings grouped by severity.
type WarningSeverityCounts struct {
	Info     int
	Warning  int
	Error    int
	Blocking int
}

// Total returns the sum of all severity counts.
func (c WarningSeverityCounts) Total() int {
	return c.Info + c.Warning + c.Error + c.Blocking
}
