package common

import (
	"gopkg.in/gcfg.v1"
)

// SoConfiguration struct for configuration data
type SoConfiguration struct {
	Dgraph struct {
		Host string
		Port int
		CloutDBType string
	}
}

// ConfigFile configuration file
const ConfigFile = "../config/application.cfg"

// SoConfig global variable to store the configuration
var SoConfig SoConfiguration

// Initialize global variable for configuration
func init(){
	// struct to hold SO configuration
	SoConfig = SoConfiguration{}
	// read configuration from a file
	err := gcfg.ReadFileInto(&SoConfig, ConfigFile)
	if err != nil {
		FailOnError(err)
	}
}