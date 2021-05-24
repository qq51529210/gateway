package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func loadConfig() *NewGatewayData {
	var cfg NewGatewayData
	var data []byte
	var err error
	// First arg is local file path or http url.
	if len(os.Args) > 1 {
		// Configure from http server.
		if strings.HasPrefix(os.Args[1], "http://") || strings.HasPrefix(os.Args[1], "https://") {
			res, err := http.Get(os.Args[1])
			if err != nil {
				panic(err)
			}
			defer res.Body.Close()
			data, err = ioutil.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}
		} else {
			data, err = ioutil.ReadFile(os.Args[1])
			if err != nil {
				panic(err)
			}
		}
	} else {
		// No arg,use "appname.json" as configure file.
		dir, file := filepath.Split(os.Args[0])
		ext := filepath.Ext(file)
		data, err = ioutil.ReadFile(filepath.Join(dir, file[:len(file)-len(ext)]+".json"))
		if err != nil {
			panic(err)
		}
	}
	// Parse json.
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}

func main() {
	gw, err := NewGateway(loadConfig())
	if err != nil {
		panic(err)
	}
	gw.Serve()
}
