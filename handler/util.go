package handler

// reurn dir top name.
// Example: "/a/b" return "/a".
func TopDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}
