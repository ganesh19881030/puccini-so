package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbInterfaceImplementation struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbInterfaceImplementation) Convert(responseData *interface{}, readerName string) interface{} {
	return AddReaderNameToData(responseData, readerName)
}

func (ntemp *DbInterfaceImplementation) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	var compMap ard.Map
	var err error
	var name string

	name, err = ExtractNameFromFieldData(fieldData)
	common.FailOnError(err)

	compMap, err = dgt.FindNestedComp(uid, key, name)
	common.FailOnError(err)

	return compMap
}
func (ntemp *DbInterfaceImplementation) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
