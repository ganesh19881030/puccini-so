package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbTriggerDefinitionCondition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbTriggerDefinitionCondition) Convert(responseData *interface{}, readerName string) interface{} {
	return TransformConditionData(*responseData)
}

func (ntemp *DbTriggerDefinitionCondition) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	name, err := ExtractNameFromFieldData(fieldData)
	common.FailOnError(err)
	compMap, err := dgt.FindConditionComps(uid, key, name)
	common.FailOnError(err)
	return compMap
}

func (ntemp *DbTriggerDefinitionCondition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
