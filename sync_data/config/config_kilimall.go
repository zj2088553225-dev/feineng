package config

type Kilimall struct {
	Cookie       string `yaml:"cookie"`
	AuthToken    string `yaml:"auth_token"`
	BaseURL      string `yaml:"base_url"`
	LogisticsAPI string `yaml:"logistics_api"`
	RegionID     int    `yaml:"region_id"`
	RegionCode   string `yaml:"region_code"`
	OrderStatus  int    `yaml:"order_status"`
	ReturnSkus   int    `yaml:"return_skus"`
	TimeType     int    `yaml:"time_type"`
	StartTime    string `yaml:"start_time"`
	EndTime      string `yaml:"end_time"`
	PageSize     int    `yaml:"page_size"`
	MaxRetries   int    `yaml:"max_retries"`
	DelayMs      int    `yaml:"delay_ms"`
}
