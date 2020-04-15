package common

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"gopkg.in/gcfg.v1"
)

// SoConfiguration struct for configuration data
type SoConfiguration struct {
	Dgraph struct {
		Host           string
		Port           int
		Ctype          string
		CldbType       CloutDbType
		SchemaFilePath string
	}

	Remote struct {
		RemoteHost   string
		RemotePort   int
		RemoteUser   string
		RemotePubKey string
	}
}

// ConfigFile configuration file
const ConfigFile = "config/application.cfg"

// CloutDbType to select how Clout is persisted in Dgraph
type CloutDbType int

const (
	// Translated form of clout where props and attribs are stored
	// as blobs (json strings) in Dgraph
	Translated CloutDbType = iota + 1
	// Original Clout structure with minimal changes (minus javascript)
	Original
	// Refined in terms of reusable TOSCA entities like node types, data types, etc.
	Refined
)

var CloutDbTypeMap map[string]CloutDbType

var lock = &sync.Mutex{}

// SoConfig global variable to store the configuration
var SoConfig *SoConfiguration

// Initialize global variable for configuration
func init() {

	CloutDbTypeMap = make(map[string]CloutDbType)
	CloutDbTypeMap["original"] = Original
	CloutDbTypeMap["translated"] = Translated
	CloutDbTypeMap["refined"] = Refined

}

func ReadConfiguration() {

	if SoConfig == nil {
		lock.Lock()
		defer lock.Unlock()
		if SoConfig == nil {

			// struct to hold SO configuration
			SoConfig = &SoConfiguration{}
			// read configuration from a file
			var err error
			var count int
			var cnf string
			prefix := ""
			cnf = filepath.Join(".", ConfigFile)
			for {
				err = gcfg.ReadFileInto(SoConfig, cnf)
				if err == nil || count > 4 {
					break
				} else {
					fmt.Println("Config file error: ", err)
					abspath, _ := filepath.Abs(cnf)
					fmt.Println("Failed to read config file at ", abspath)
				}
				prefix = filepath.Join("..", prefix)
				cnf = filepath.Join(prefix, ConfigFile)
				count++
			}
			if err != nil {
				FailOnError(err)
			}

			SoConfig.Dgraph.CldbType = CloutDbTypeMap[SoConfig.Dgraph.Ctype]
			if SoConfig.Dgraph.CldbType < Translated ||
				SoConfig.Dgraph.CldbType > Refined {

				err = errors.New("Invalid configuration - ctype - defined in dgraph section of application.cfg")
				FailOnError(err)
			}
		}
	}
}
