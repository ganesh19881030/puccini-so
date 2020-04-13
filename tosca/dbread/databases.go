package dbread

import (
	"github.com/tliron/puccini/tosca/dbread/dgraph"
	"github.com/tliron/puccini/tosca/dbread/dgraph/db_tosca_v1_3"
)

var DbGrammar dgraph.DbGrammarMap = make(dgraph.DbGrammarMap)

func init() {
	DbGrammar["dgraph.tosca_v1_3"] = db_tosca_v1_3.DbObjectMap
}
