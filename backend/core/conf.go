package core

import (
	"backend/config"
	"backend/global"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// const configFile = "D:\\learn\\jumia\\sync_data\\settings.yaml"
const configFile = "../sync_data/settings.yaml"

// 保证两个程序从同一个文件中读取配置信息就可以了，这个程序不需要更新token,在另一个程序中更新过了
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
