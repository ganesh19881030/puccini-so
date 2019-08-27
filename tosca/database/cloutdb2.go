package database

import (
	"fmt"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/tosca/normal"
)

// CloutDB2 defines an implementation of CloutDB
type CloutDB2 struct {
	Dburl string
	clout bool
}

// NewCloutDb2 creates a CloutDB2 instance
func NewCloutDb2(dburl string) CloutDB {
	return CloutDB2{dburl, true}
}

// SaveServiceTemplate method implementation of CloutDB interface for CloutDB2 instance
func (db CloutDB2) SaveServiceTemplate(s *normal.ServiceTemplate, urlString string, grammarVersion string, internalImport string) error {
	return nil
}

// IsCloutCapable - true if it handles clout structure, false otherwise
func (db CloutDB2) IsCloutCapable() bool {
	return db.clout
}

// SaveClout method implementation of CloutDB interface for CloutDB2 instance
func (db CloutDB2) SaveClout(clout *clout.Clout, urlString string, grammarVersions string, internalImport string) error {
	var printout = true
	var dgraphset = DgraphSet{}

	var cloutItem = make(ard.Map)

	clout.Metadata["puccini-js"] = ""

	cloutItem["clout:data"] = clout

	topologyName := extractTopologyName(urlString)
	cloutItem["clout:name"] = topologyName
	cloutItem["clout:version"] = clout.Version
	cloutItem["clout:grammarversion"] = grammarVersions

	dgraphset.Set = append(dgraphset.Set, cloutItem)

	// write out the Dgraph data in JSON format
	if printout {
		err := format.WriteOrPrint(dgraphset, "json", true, "")
		common.FailOnError(err)
		fmt.Println("-")
		fmt.Println("---------------------------------------------------")
		fmt.Println("-")
	}

	// save clout data into Dgraph
	SaveCloutGraph(&dgraphset, db.Dburl)
	return nil
}
