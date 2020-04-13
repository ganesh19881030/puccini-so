package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbAttributeMapping struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbAttributeMapping) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	mapData := make(ard.Map)
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			var amap ard.List
			if capname, ok := data.(ard.Map)["capabilityname"]; ok {
				amap = make(ard.List, 3)
				amap[0] = data.(ard.Map)["nodetemplatename"]
				amap[1] = capname
				amap[2] = data.(ard.Map)["attributename"]

			} else {
				amap = make(ard.List, 2)
				amap[0] = data.(ard.Map)["nodetemplatename"]
				amap[1] = data.(ard.Map)["attributename"]
			}
			mapData[mkey] = amap
		}
	}
	return mapData
}

func (ntemp *DbAttributeMapping) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}
func (ntemp *DbAttributeMapping) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
