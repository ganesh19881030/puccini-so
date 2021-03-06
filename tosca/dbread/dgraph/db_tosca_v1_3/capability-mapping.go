package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbCapabilityMapping struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbCapabilityMapping) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	rmapData := make(ard.Map)
	mapData := ConvertDbResponseArrayToMap(responseData, readername)
	for mkey1, mval1 := range mapData {
		var vList ard.List
		vList = append(vList, mval1.(ard.Map)["nodetemplatename"])
		vList = append(vList, mval1.(ard.Map)["capabilityname"])
		rmapData[mkey1] = vList
	}
	return rmapData
	//return ConvertDbResponseArrayToMap(responseData, readername)
}

func (ntemp *DbCapabilityMapping) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbCapabilityMapping) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
