package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbAttributeValue struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbAttributeValue) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	mapData := make(ard.Map)
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			/*amap := make(ard.Map)
			amap[data.(ard.Map)["functionname"].(string)] = data.(ard.Map)["fnarguments"]
			mapData[mkey] = amap*/
			mapData[mkey] = TransformValueData(data)

		}
	}

	return mapData
}

func (ntemp *DbAttributeValue) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindAttributeComps(uid)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbAttributeValue) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
