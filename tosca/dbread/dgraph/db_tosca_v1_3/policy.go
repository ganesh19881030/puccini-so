package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

func init() {
	dgraph.FieldRegistries.ArrayRegistry["DirectivesTargetNodeTemplateOrGroupNames"] = true
}

type DbPolicy struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbPolicy) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	// no conversion needed
	return ConvertDbResponseArrayToSequencedList(*responseData, readername)
}

func (ntemp *DbPolicy) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindPolicyComp(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbPolicy) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
