package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

func init() {
	dgraph.FieldRegistries.MergeRegistry["PolicyType.Type"] = true
	dgraph.FieldRegistries.ArrayRegistry["TargetNodeTypeOrGroupTypeNames"] = true
}

type DbPolicyType struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbPolicyType) Convert(responseData *interface{}, readerName string) interface{} {
	return AddReaderNameToData(responseData, readerName)
}
func (ntemp *DbPolicyType) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	compMap, err := dgt.FindPolicyTypeComp(uid, key)
	common.FailOnError(err)
	return compMap
}

func (ntemp *DbPolicyType) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
