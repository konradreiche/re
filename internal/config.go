package re

import "os"

type Config struct {
	RESTEndpoint string
	Endpoint     string
	AccessToken  string
}

func NewConfig() Config {
	endpoint := "https://api.github.com"
	restEndpoint := "https://api.github.com"
	accessToken := os.Getenv("GH_TOKEN")
	if ghe := os.Getenv("GITHUB_ENTERPRISE_URL"); ghe != "" {
		restEndpoint = ghe + "/api/v3"
		endpoint = ghe + "/api"
		accessToken = os.Getenv("GH_ENTERPRISE_TOKEN")
	}
	return Config{
		RESTEndpoint: restEndpoint,
		Endpoint:     endpoint,
		AccessToken:  accessToken,
	}
}
