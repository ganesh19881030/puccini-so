package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbRequirementAssignment struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbRequirementAssignment) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	var childDataMod []interface{}
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		cnstrMap := make(ard.Map)
		nam := data.(ard.Map)["name"].(string)
		xMap := make(ard.Map)
		ttntn := false
		for dkey, str := range data.(ard.Map) {
			if !(dkey == "name" ||
				//dkey == "uid" ||
				//dkey == "readername" ||
				dkey == "namespace") {
				xMap[dkey] = str
			}
			if dkey == "targetnodetemplatenameortypename" {
				ttntn = true
			}
		}
		if len(xMap) == 3 && ttntn {
			cnstrMap[nam] = xMap["targetnodetemplatenameortypename"]
		} else {
			cnstrMap[nam] = xMap
		}
		childDataMod = append(childDataMod, cnstrMap)
	}
	return childDataMod
}

func (ntemp *DbRequirementAssignment) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbRequirementAssignment) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
