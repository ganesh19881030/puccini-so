package db_tosca_v1_3

import (
	"errors"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

func init() {
	dgraph.FieldRegistries.MergeRegistry["CapabilityType.Type"] = true
}

type DbCapabilityType struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbCapabilityType) Convert(responseData *interface{}, readerName string) interface{} {
	return AddReaderNameToData(responseData, readerName)
}

func (ntemp *DbCapabilityType) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {
	return ConvertDbResponseArrayToMap(responseData, readername)
}

func (ntemp *DbCapabilityType) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	var compMap ard.Map
	var err error
	var name string
	if key == "validtargetcapabilitytypes" {
		compMap, err = dgt.FindNestedComps(uid, key)
	} else {
		name, err = ExtractNameFromFieldData(fieldData)
		common.FailOnError(err)
		compMap, err = dgt.FindNestedComp(uid, key, name)
	}
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbCapabilityType) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
