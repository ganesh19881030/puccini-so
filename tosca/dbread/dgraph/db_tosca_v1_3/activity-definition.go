package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbActivityDefinition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbActivityDefinition) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	var aList ard.List
	for _, adata := range *responseData {
		amap := make(ard.Map)
		for akey, aval := range adata.(ard.Map) {
			if !( //akey == "uid" ||
			akey == "name" ||
				akey == "namespace" ||
				akey == "readername") {
				amap[akey] = aval
			}
		}
		aList = append(aList, amap)
	}
	return aList

}

func (ntemp *DbActivityDefinition) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbActivityDefinition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
