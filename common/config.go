package common

import (
	"errors"

	"gopkg.in/gcfg.v1"
)

// SoConfiguration struct for configuration data
type SoConfiguration struct {
	Dgraph struct {
		Host     string
		Port     int
		Ctype    string
		CldbType CloutDbType
	}
}

// ConfigFile configuration file
const ConfigFile = "../config/application.cfg"

// CloutDbType to select how Clout is persisted in Dgraph
type CloutDbType int

const (
	// Translated form of clout where props and attribs are stored
	// as blobs (json strings) in Dgraph
	Translated CloutDbType = iota + 1
	// Original Clout structure with minimal changes (minus javascript)
	Original
)

var CloutDbTypeMap map[string]CloutDbType

// SoConfig global variable to store the configuration
var SoConfig SoConfiguration

// Initialize global variable for configuration
func init() {

	CloutDbTypeMap = make(map[string]CloutDbType)
	CloutDbTypeMap["original"] = Original
	CloutDbTypeMap["translated"] = Translated

	// struct to hold SO configuration
	SoConfig = SoConfiguration{}
	// read configuration from a file
	err := gcfg.ReadFileInto(&SoConfig, ConfigFile)
	if err != nil {
		FailOnError(err)
	}
	SoConfig.Dgraph.CldbType = CloutDbTypeMap[SoConfig.Dgraph.Ctype]
	if SoConfig.Dgraph.CldbType < Translated ||
		SoConfig.Dgraph.CldbType > Original {

		err = errors.New("Invalid configuration - ctype - defined in dgraph section of application.cfg")
		FailOnError(err)
	}
}
