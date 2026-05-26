package filesystem

func shouldIgnoreSkillEntryName(name string) bool {
	return name == ".DS_Store"
}
