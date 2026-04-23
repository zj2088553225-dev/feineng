package config

type Jumia struct {
	ClientId     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	GrantType    string `yaml:"grant_type"`
	RedirectUrl  string `yaml:"redirect_url"`
	RefreshToken string `yaml:"refresh_token"`
	AccessToken  string `yaml:"access_token"`
}
