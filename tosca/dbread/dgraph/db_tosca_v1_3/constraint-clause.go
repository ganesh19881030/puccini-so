package db_tosca_v1_3

import (
	"errors"
	"strconv"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

//
// ConstraintClause
//
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.3
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.2
//

type DbConstraintClause struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbConstraintClause) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {

	var childDataMod []interface{}
	cnstrMap := make(ard.Map)
	for _, data := range *responseData {
		op := data.(ard.Map)["operator"].(string)
		//if op == "in_range" || op == "greater_or_equal" {
		//	fmt.Println("prop-filter/const-clause ", op)
		//}
		argstr := data.(ard.Map)["arguments"].(string)
		var argList []interface{}
		var err error
		dtname, ok := (*contextData).(ard.Map)["datatypename"].(string)
		if strings.HasPrefix(dtname, "scalar-unit.") {
			argList, err = ParseScalarUnitArguments(argstr)
		} else {
			argList, err = ParseArguments(argstr)
		}
		common.FailOnError(err)
		if op == "in_range" || (ok && (dtname == "integer" || dtname == "range")) {
			argl := make(ard.List, len(argList))
			for ind, arg := range argList {
				intarg, err := strconv.Atoi(arg.(string))
				argl[ind] = intarg
				common.FailOnError(err)
			}
			cnstrMap[op] = argl
		} else if ok && dtname == "boolean" {
			argl := make(ard.List, len(argList))
			for ind, arg := range argList {
				boolarg, err := strconv.ParseBool(arg.(string))
				argl[ind] = boolarg
				common.FailOnError(err)
			}
			cnstrMap[op] = argl
		} else {
			cnstrMap[op] = argList
		}
		childDataMod = append(childDataMod, cnstrMap)
	}
	return childDataMod

}

func (ntemp *DbConstraintClause) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbConstraintClause) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
