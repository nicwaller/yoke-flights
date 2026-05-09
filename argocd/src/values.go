package main

type Values struct {
	Server         bool `yaml:"server"`
	ServerInsecure bool `yaml:"serverInsecure"`
}

var defaults = Values{
	Server: true,
}
