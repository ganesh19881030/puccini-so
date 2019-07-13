package database

import (
	"fmt"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/clout"

)

// CloutDB interface for defining the db methods
type CloutDB interface {
	Save(clout *clout.Clout, urlString string, grammarVersion string) error
}

// CreateCloutDBClient is the factory method to create the appropriate CloutDB 
// implementation
func CreateCloutDBClient () CloutDB {

	// construct Dgraph url from configuration
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)
	
	switch common.SoConfig.Dgraph.CloutDBType {
	case "1":
		return NewCloutDb1(dburl)
		break
	case "2":
		return NewCloutDb2(dburl)
		break
	default:
		return NewCloutDb1(dburl)
	}

	return nil

}