package main

type Values struct {
	Server            bool   `yaml:"server"`
	ServerInsecure    bool   `yaml:"serverInsecure"`
	ServerServiceType string `yaml:"serverServiceType"`
	ServerPort        int32  `yaml:"serverPort"`
}

var defaults = Values{
	Server:            true,
	ServerServiceType: "ClusterIP",
	ServerPort:        5000,
}
