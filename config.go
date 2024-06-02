package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type (
	vpn struct {
		OpenVPN   string
		Sudo      string
		Shell     string
		ShellArgs []string
	}

	server struct {
		Addr string
	}

	config struct {
		Debug   bool
		Browser bool
		Vpn     vpn
		Server  server
	}
)

func loadConfig(filename string) (c *config, err error) {
	fileBytes, err := os.ReadFile(filename)

	if err != nil {
		return
	}

	c = &config{}
	err = yaml.Unmarshal(fileBytes, c)

	return
}
