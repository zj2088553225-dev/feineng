package config

// 存储配置信息结构体
type Config struct {
	MySQL    MySQL    `yaml:"mysql"`
	System   System   `yaml:"system"`
	Logger   Logger   `yaml:"logger"`
	Jumia    Jumia    `yaml:"jumia"`
	JumiaTwo Jumia    `yaml:"jumia_two"`
	Jwy      Jwy      `yaml:"jwy"`
	Wuliu    Wuliu    `yaml:"wuliu"`
	Kilimall Kilimall `yaml:"kilimall"`
}
