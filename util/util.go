package util

// 返回第一层目录名称，"/a/b"中的"/a"
func TopDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}

// func SortHandlerData(data []*handler.NewHandlerData) []*handler.NewHandlerData {
// 	for i := 0; i < len(data); i++ {
// 		for j := i + 1; j < len(data); j++ {
// 			if data[j].Sort < data[i].Sort {
// 				d := data[j]
// 				data[j] = data[i]
// 				data[i] = d
// 			}
// 		}
// 	}
// 	return data
// }
