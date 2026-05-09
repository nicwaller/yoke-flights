package main

import "fmt"

type Values struct {
	Token string `yaml:"token"`
	Image string `yaml:"image"`
}

var defaults = Values{
	Image: "cloudflare/cloudflared:latest",
}

func (v Values) validate() error {
	if v.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}
