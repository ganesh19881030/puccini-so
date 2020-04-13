package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

//
// ServiceTemplate
//
type DbServiceTemplate struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbServiceTemplate) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	var compMap ard.Map
	name := uid
	urlstring := key
	fnd, stuid, grammarVersion, err := dgt.FindCompByTypeAndNamespace("ServiceTemplate", name, urlstring)
	if err == nil && fnd {
		log.Debugf("ServiceTemplate uid: %s", stuid)
		compMap, err = dgt.FindCompByUid(stuid)
		common.FailOnError(err)
		if err == nil && len(compMap) > 0 {
			//compMap["tosca_definitions_version"] = "tosca_simple_yaml_1_3"
			v1 := compMap["comp"].([]interface{})
			v2 := v1[0].(map[string]interface{})
			v2["toscadefinitionsversion"] = grammarVersion
			v2["tosca_definitions_version"] = grammarVersion
			v2["uid"] = stuid
			compMap = v2
		}
	}
	return compMap
}

func (ntemp *DbServiceTemplate) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
