package core

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"sync_data/config"
	"sync_data/global"
)

const configFile = "settings.yaml"

// 读取配置操作
// 读取yaml文件的配置
func InitConf() {
	c := &config.Config{}
	yamlConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(fmt.Errorf("read config file error, %v", err))
	}
	err = yaml.Unmarshal(yamlConfig, c)
	if err != nil {
		log.Fatal("unmarshal config file error, %v", err)
	}
	log.Println("config yamlFile load Init success.")
	global.Config = c
	//测试配置信息
	//log.Println(global.Config.Jumia)
}

func SetYaml() error {

	byteData, err := yaml.Marshal(global.Config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configFile, byteData, fs.ModePerm)
	if err != nil {
		return err
	}
	global.Log.Info("配置文件修改成功")
	return nil
}
