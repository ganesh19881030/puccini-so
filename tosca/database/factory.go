package database

import (
	"fmt"

	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
)

// CloutDB interface for defining the db methods
type CloutDB interface {
	Save(clout *clout.Clout, urlString string, grammarVersion string, internalImport string) error
}

// CreateCloutDBClient is the factory method to create the appropriate CloutDB
// implementation
func CreateCloutDBClient() CloutDB {

	// construct Dgraph url from configuration
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)

	switch common.SoConfig.Dgraph.CldbType {
	case common.Translated:
		return NewCloutDb1(dburl)
	case common.Original:
		return NewCloutDb2(dburl)
	default:
		return NewCloutDb1(dburl)
	}

}
