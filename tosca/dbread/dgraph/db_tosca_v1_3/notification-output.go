package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbNotificationOutput struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbNotificationOutput) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	var ok bool
	var dmap ard.Map
	var nodeTemplateName, attributeName, mkey string
	var list ard.List

	mapData := make(ard.Map)
	for _, data := range *responseData {
		dmap, ok = data.(ard.Map)
		if ok {
			mkey, ok = dmap["name"].(string)
			if ok {
				nodeTemplateName, ok = dmap["nodetemplatename"].(string)
				if ok {
					attributeName, ok = dmap["attributename"].(string)
					if ok {
						list = make(ard.List, 2)
						list[0] = nodeTemplateName
						list[1] = attributeName
					}
				}
			}
		}
		if ok {
			mapData[mkey] = list
		}
	}
	return mapData
}

func (ntemp *DbNotificationOutput) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbNotificationOutput) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
