package gateway

import "fmt"

// 找出第一层目录，/a/b中的/a
func TopDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}

// data必须有key为name，value为string类型的数据。
func MustGetString(data map[string]interface{}, name string) (string, error) {
	val, ok := data[name]
	if !ok {
		return "", fmt.Errorf(`"%s" must be defined`, name)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf(`"%s" must be "string" type`, name)
	}
	return str, nil
}

// data如果有key为name的数据，那么value必须string类型。
func GetString(data map[string]interface{}, name string) (string, error) {
	val, ok := data[name]
	if ok {
		str, ok := val.(string)
		if !ok {
			return "", fmt.Errorf(`"%s" must be "string" type`, name)
		}
		return str, nil
	}
	return "", nil
}

func FilteNilHandler(handler ...Handler) []Handler {
	hd := make([]Handler, 0)
	for i := 0; i < len(handler); i++ {
		if handler[i] != nil {
			hd = append(hd, handler[i])
		}
	}
	return hd
}
