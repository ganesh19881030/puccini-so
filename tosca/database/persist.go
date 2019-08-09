package database

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
)

// DgraphSet for Dgraph json data
type DgraphSet struct {
	Set []ard.Map `json:"set"`
}

// Persist clout data
func Persist(clout *clout.Clout, urlString string, grammarVersions []string, internalImport string) error {

	cloutdbClient := CreateCloutDBClient()
	var versions string
	for ind, val := range grammarVersions {
		if ind > 0 {
			versions = versions + ","
		}
		versions = versions + val
	}

	err := cloutdbClient.Save(clout, urlString, versions, internalImport)

	return err
}
