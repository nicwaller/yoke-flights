package main

type Values struct {
	Server bool `yaml:"server"`
}

var defaults = Values{
	Server: true,
}
