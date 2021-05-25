package handler

import "encoding/json"

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

// Convert map to struct,in a simple but stupid way.
// Arg s is a pointer.
func Map2Struct(m map[string]interface{}, s interface{}) error {
	// First,convert map to json.
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	// Second,convert json to struct.
	return json.Unmarshal(data, s)
}
