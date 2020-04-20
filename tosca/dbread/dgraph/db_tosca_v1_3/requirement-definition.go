package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbRequirementDefinition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbRequirementDefinition) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	var xdata []interface{}
	var xmap ard.Map
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			xmap = make(ard.Map)
			xmap[mkey] = data
			xdata = append(xdata, xmap)
		}
	}
	return xdata
}

func (ntemp *DbRequirementDefinition) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbRequirementDefinition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
