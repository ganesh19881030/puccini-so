package common

import (
	"gopkg.in/gcfg.v1"
)

// SoConfig struct for configuration data
type SoConfig struct {
	Dgraph struct {
		Host string
		Port int
	}
}

// CONFIG_FILE configuration file
var CONFIG_FILE = "../config/application.cfg"

// GetSoConfig fetches the application configuration data from a file
func GetSoConfig() (SoConfig, error) {
	// struct to hold SO configuration
	socfg := SoConfig{}
	// read configuration from a file
	err := gcfg.ReadFileInto(&socfg, CONFIG_FILE)

	return socfg, err
}
