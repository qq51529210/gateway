package handler

// 返回第一层目录名称，"/a/b"中的"/a"
func TopDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}
