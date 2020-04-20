package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

//
// ConditionClause
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.25
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.21
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.19
//

type DbRequirementMapping struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbRequirementMapping) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	rmapData := make(ard.Map)
	mapData := ConvertDbResponseArrayToMap(responseData, readername)
	for mkey1, mval1 := range mapData {
		var vList ard.List
		vList = append(vList, mval1.(ard.Map)["nodetemplatename"])
		vList = append(vList, mval1.(ard.Map)["requirementname"])
		rmapData[mkey1] = vList
	}
	return rmapData
}

func (ntemp *DbRequirementMapping) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbRequirementMapping) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
