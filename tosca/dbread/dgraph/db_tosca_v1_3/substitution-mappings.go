package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbSubstitutionMappings struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbSubstitutionMappings) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	// no conversion needed
	return *responseData
}

func (ntemp *DbSubstitutionMappings) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbSubstitutionMappings) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, string, error) {
	obj, ok := searchObject.(dgraph.SearchFields)
	if ok {
		name, _ := GetFieldString(obj.EntityPtr, "NodeTypeName")
		fnd, uid, err := dgt.FindComp(name, obj.ObjectDGType, obj.ObjectNSuid, obj.SubjectUid, obj.Predicate)
		return fnd, uid, name, err
	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DbFind function."))
	}
	return false, "", obj.ObjectKey, nil
}

func (ntemp *DbSubstitutionMappings) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
