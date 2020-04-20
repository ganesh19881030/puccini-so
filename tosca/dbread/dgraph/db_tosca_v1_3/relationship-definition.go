package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbRelationshipDefinition struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbRelationshipDefinition) Convert(responseData *interface{}, readerName string) interface{} {
	return AddReaderNameToData(responseData, readerName)
}

func (ntemp *DbRelationshipDefinition) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	return ConvertDbResponseArrayToMap(responseData, readername)
}

func (ntemp *DbRelationshipDefinition) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	var compMap ard.Map
	var err error
	var name string

	name, err = ExtractNameFromFieldData(fieldData)
	common.FailOnError(err)
	compMap, err = dgt.FindNestedComp(uid, key, name)

	common.FailOnError(err)

	return compMap
}

func (ntemp *DbRelationshipDefinition) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, string, error) {
	obj, ok := searchObject.(dgraph.SearchFields)
	if ok {
		fnd, uid, err := dgt.FindComp(obj.Mapkey, obj.ObjectDGType, obj.ObjectNSuid, obj.SubjectUid, obj.Predicate)
		return fnd, uid, obj.Mapkey, err
	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DbFind function."))
	}
	return false, "", obj.Mapkey, nil
}

func (ntemp *DbRelationshipDefinition) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
