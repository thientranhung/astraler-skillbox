package filesystem

func shouldIgnoreSkillEntryName(name string) bool {
	switch name {
	case ".DS_Store", ".gitkeep":
		return true
	}
	return false
}
