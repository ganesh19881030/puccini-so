package database

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/tosca/normal"
)

// DgraphSet for Dgraph json data
type DgraphSet struct {
	Set []ard.Map `json:"set"`
}

// Persist clout data
func Persist(clout *clout.Clout, s *normal.ServiceTemplate, urlString string, grammarVersions []string, internalImport string) error {

	cloutdbClient := CreateCloutDBClient()
	var versions string
	for ind, val := range grammarVersions {
		if ind > 0 {
			versions = versions + ","
		}
		versions = versions + val
	}

	var err error
	if cloutdbClient.IsCloutCapable() {
		err = cloutdbClient.SaveClout(clout, urlString, versions, internalImport)
	} else {
		err = cloutdbClient.SaveServiceTemplate(s, urlString, versions, internalImport)

	}

	return err
}
