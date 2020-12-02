package config

import (
	"flag"
	"io/ioutil"

	"github.com/jessevdk/go-assets"
	"gopkg.in/yaml.v2"
)

var configPath = "/config/"

type Config struct {
	Env string
	Map *map[string]interface{}
}

var GlobalConfig *Config = new(Config)

func (config *Config) readConfig() {
	t1 := readConfigFile(configPath + "config.yml")
	t2 := readConfigFile(configPath + "config-" + config.Env + ".yml")
	for k, v := range *t2 {
		(*t1)[k] = v
	}
	if (*t1)["external"] != nil && (*t1)["external"] != "" {
		s, _ := (*t1)["external"].(string)
		t3 := readExternalFile(s)
		for k, v := range *t3 {
			(*t1)[k] = v
		}
	}
	config.Map = t1
}

func readConfigFile(path string) *map[string]interface{} {
	file, _ := localAssets.Open(path)
	data, _ := ioutil.ReadAll(file)
	// data, _ := ioutil.ReadFile(path)
	t := map[string]interface{}{}
	_ = yaml.Unmarshal(data, &t)
	return &t
}

func readExternalFile(path string) *map[string]interface{} {
	data, _ := ioutil.ReadFile(path)
	t := map[string]interface{}{}
	_ = yaml.Unmarshal(data, &t)
	return &t
}

var localAssets *assets.FileSystem

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
