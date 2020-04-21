package db_tosca_v1_3

import (
	"encoding/json"
	"errors"
	"fmt"

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

func (ntemp *DbRequirementAssignment) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, string, error) {
	obj, ok := searchObject.(dgraph.SearchFields)
	if ok {
		query2 := `
	query {
	    comp(func: uid(<%s>))@cascade  {
		%s @filter(eq(name,"%s") %s %s){
		  	uid
		  	name
			}
		}
	}`
		var qp1, qp2 string
		querypart1 := `and eq(targetnodetemplatenameortypename,"%s")`
		querypart2 := `and eq(targetcapabilitynameortypename,"%s")`
		var ntnamestr, capnamestr string
		ntnamestr, _ = GetFieldString(obj.EntityPtr, "TargetNodeTemplateNameOrTypeName")
		if ntnamestr != "" {
			qp1 = fmt.Sprintf(querypart1, ntnamestr)
		}
		capnamestr, _ = GetFieldString(obj.EntityPtr, "TargetCapabilityNameOrTypeName")
		if capnamestr != "" {
			qp2 = fmt.Sprintf(querypart2, capnamestr)
		}

		nquad := fmt.Sprintf(query2, obj.SubjectUid, obj.Predicate, obj.ObjectKey, qp1, qp2)
		log.Debugf("nquad: %s", nquad)

		resp, err := dgt.ExecQuery(nquad)

		var uid string
		var fnd bool
		if err == nil {
			var a interface{}
			err = json.Unmarshal(resp.Json, &a)
			if err == nil {
				b := a.(map[string]interface{})
				uidlist := b["comp"].([]interface{})
				if len(uidlist) > 0 {
					c := uidlist[0].(map[string]interface{})
					d := c[obj.Predicate].(interface{})
					var e map[string]interface{}
					_, ok := d.([]interface{})
					if ok {
						d1 := d.([]interface{})
						e = d1[0].(map[string]interface{})
					} else {
						e, ok = d.(map[string]interface{})
					}
					if ok {
						uid = e["uid"].(string)
						fnd = len(uid) > 0
					}
				}
			}

		}

		return fnd, uid, obj.ObjectKey, err
		//return dgt.FindComp(obj.ObjectKey, obj.ObjectDGType, obj.ObjectNSuid, obj.SubjectUid, obj.Predicate)
	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DbFind function."))
	}
	return false, "", obj.ObjectKey, nil
}

func (ntemp *DbRequirementAssignment) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
