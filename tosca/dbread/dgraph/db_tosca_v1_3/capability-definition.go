package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbCapabilityDefinition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbCapabilityDefinition) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	return ConvertDbResponseArrayToMap(responseData, readername)
}

func (ntemp *DbCapabilityDefinition) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {

	return ConvertDbResponseArrayToMap(responseData, readername)

}

func (ntemp *DbCapabilityDefinition) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbCapabilityDefinition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
