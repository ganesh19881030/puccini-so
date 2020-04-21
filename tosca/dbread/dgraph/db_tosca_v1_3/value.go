package db_tosca_v1_3

import (
	"errors"
	"strconv"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbValue struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbValue) Convert(responseData *interface{}, readerName string) interface{} {
	return TransformValueData(*responseData)
}

func (ntemp *DbValue) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	mapData := make(ard.Map)
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			if key == "inputs" {
				/*amap := make(ard.Map)
				amap[data.(ard.Map)["functionname"].(string)] = data.(ard.Map)["fnarguments"]
				mapData[mkey] = amap*/
				mapData[mkey] = TransformValueData(data)
			} else if key == "properties" || key == "propertymappings" {
				mapData[mkey] = TransformValueData(data)
			}
		}
	}

	return mapData
}

func (ntemp *DbValue) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	var compMap ard.Map
	var err error
	if key == "attributes" {
		compMap, err = dgt.FindAttributeComps(uid)
	} else if key == "inputs" ||
		key == "properties" ||
		key == "propertymappings" ||
		key == "value" {
		compMap, err = dgt.FindValueComp(uid, key)
	}
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbValue) ByPassDbRead(contextData *interface{}, name string, key string) bool {
	if key == "default" {
		dtypei, ok := (*contextData).(ard.Map)["datatypename"]
		childData := (*contextData).(ard.Map)[key]
		if ok {
			dtype := dtypei.(string)
			var sdata string
			sdata, ok = childData.(string)
			if ok {
				if dtype == "integer" {
					idata, err := strconv.Atoi(sdata)
					common.FailOnError(err)
					childData = idata
				} else if dtype == "boolean" {
					bdata, err := strconv.ParseBool(sdata)
					common.FailOnError(err)
					childData = bdata
				} else if dtype == "float" {
					fdata, err := strconv.ParseFloat(sdata, 32)
					common.FailOnError(err)
					childData = fdata
				} else if dtype != "string" {
					log.Debugf("*** unhandled data type: %s", dtype)
				}
				(*contextData).(ard.Map)[key] = childData
			}
		} else {
			log.Debugf("*** default data type name is undefined for: %s", name)
		}

		return true
	}
	return false
}

func (ntemp *DbValue) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, string, error) {

	obj, ok := searchObject.(dgraph.SearchFields)
	if ok {
		if obj.Predicate == "default" {
			// may need it in the future - leaving it in here for now until we have
			// enough models persisted
			//
			//AddFieldToComponent(entityPtr, bag)
			return true, obj.SubjectUid, obj.ObjectKey, nil
		} else {
			fnd, uid, err := dgt.FindComp(obj.ObjectKey, obj.ObjectDGType, obj.ObjectNSuid, obj.SubjectUid, obj.Predicate)
			return fnd, uid, obj.ObjectKey, err
		}
	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DbFind function."))
	}
	return false, "", obj.ObjectKey, nil
}
func (ntemp *DbValue) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		if tsmethod, ok := dgraph.ToscaMethodRegistry[dgraph.QueryValueKey]; ok {
			nquad = nquad + tsmethod.Process(&saveFields.EntityPtr, "")
		}

		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
