package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qq51529210/gateway"
)

func main() {
	// 加载配置
	cfg := make(map[string]interface{})
	// 默认程序目录下的程序名.json
	dir, file := filepath.Split(os.Args[0])
	ext := filepath.Ext(file)
	// 加载
	data, err := ioutil.ReadFile(filepath.Join(dir, file[:len(file)-len(ext)]+".json"))
	if err != nil {
		panic(err)
	}
	// 解析成json
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}
	// 创建
	gw, err := gateway.NewGateway(cfg)
	if err != nil {
		panic(err)
	}
	// 开始服务
	gw.Serve()
}
