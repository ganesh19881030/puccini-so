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
func Persist(clout *clout.Clout, urlString string, grammarVersion string) error {

	cloutdbClient := CreateCloutDBClient()

	err := cloutdbClient.Save(clout, urlString, grammarVersion)

	return err
}