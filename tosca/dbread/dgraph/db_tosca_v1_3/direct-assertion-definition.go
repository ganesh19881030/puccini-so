package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

//
// DirectAssertionDefinition
//
type DbDirectAssertionDefinition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbDirectAssertionDefinition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
