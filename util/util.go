package util

import "fmt"

// 返回第一层目录名称，"/a/b"中的"/a"
func TopDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}

type Data map[string]interface{}

// // 必须有key为name，value是string类型的数据，否则返回""和error。
func (d Data) MustString(name string) (string, error) {
	v, ok := d[name]
	if !ok {
		return "", fmt.Errorf(`"%s" must be defined`, name)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf(`"%s" must be "string" type`, name)
	}
	return s, nil
}

// data如果没有key为name的value，返回""和nil。
// data如果有key为name的数据，value不是string类型，返回""和error。
// data如果有key为name的数据，value是string类型，返回value和nil。
func (d Data) String(name string) (string, error) {
	v, ok := d[name]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf(`"%s" must be "string" type`, name)
	}
	return s, nil
}

// key为name，value是的map[string]string，否则返回error
func (d Data) StringMap(name string) (map[string]string, error) {
	data := make(map[string]string)
	v, ok := d[name]
	if !ok {
		return data, nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`"%s" must be "map[string]string" type`, name)
	}
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf(`"%s"."%s" must be "string" type`, name, k)
		}
		data[k] = s
	}
	return data, nil
}

// key为name，value是的[]string，否则返回error
func (d Data) StringSlice(name string) ([]string, error) {
	v, ok := d[name]
	if !ok {
		return nil, nil
	}
	a, ok := v.([]interface{})
	if ok {
		return nil, fmt.Errorf(`"%s" must be "[]string" type`, name)
	}
	var ss []string
	for i, v := range a {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf(`"%s[%d]" must be "string" type`, name, i)
		}
		ss = append(ss, s)
	}
	return ss, nil
}
