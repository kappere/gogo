package config

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/jessevdk/go-assets"
	"gopkg.in/yaml.v2"
)

var resourcesPath = "resources/"

type Config struct {
	Env string
	Map *map[string]interface{}
}

var GlobalConfig *Config = new(Config)

func (config *Config) readConfig() {
	t1 := readConfigFile("config.yml")
	t2 := readConfigFile("config-" + config.Env + ".yml")
	for k, v := range *t2 {
		(*t1)[k] = v
	}
	if (*t1)["external"] != nil && (*t1)["external"] != "" {
		s, _ := (*t1)["external"].(string)
		t3 := readConfigFile(s)
		for k, v := range *t3 {
			(*t1)[k] = v
		}
	}
	config.Map = t1
}

func readConfigFile(path string) *map[string]interface{} {
	data := *ReadFile(path)
	t := map[string]interface{}{}
	_ = yaml.Unmarshal(data, &t)
	return &t
}

func ReadFile(path string) *[]byte {
	if pathExists("./" + resourcesPath + path) {
		data, err := ioutil.ReadFile("./" + resourcesPath + path)
		if err != nil {
			return nil
		}
		return &data
	} else {
		file, err := localAssets.Open("/" + resourcesPath + path)
		if err != nil {
			return nil
		}
		data, _ := ioutil.ReadAll(file)
		return &data
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

var localAssets *assets.FileSystem

// GetAssets 获取资源
func GetAssets() *assets.FileSystem {
	return localAssets
}

// SetAssets 加载资源
func SetAssets(a *assets.FileSystem) {
	localAssets = a
}

func InitConfig() {
	// s.port = flag.String("port", "", "port")
	env := flag.String("env", "dev", "环境")
	flag.Parse()
	GlobalConfig.Env = *env
	GlobalConfig.readConfig()
}
