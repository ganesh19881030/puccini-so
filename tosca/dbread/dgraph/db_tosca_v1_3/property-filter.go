package db_tosca_v1_3

import (
	"errors"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbPropertyFilter struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbPropertyFilter) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	var xdata []interface{}
	var xmap ard.Map
	var err error
	for _, data := range *responseData {
		data.(ard.Map)["readername"] = readername
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			xmap = make(ard.Map)
			//var ccList []interface{}
			cList := data.(ard.Map)["constraintclauses"].([]interface{})
			cnstrMap := make(ard.Map)
			for _, cdata := range cList {
				op := cdata.(ard.Map)["operator"].(string)
				//if op == "in_range" {
				//	fmt.Println("prop-filter/const-clause ", op)
				//}
				args := cdata.(ard.Map)["arguments"].(string)
				if strings.HasPrefix(args, "[") && !strings.HasPrefix(args, "[map[") {
					cnstrMap[op], err = ParseArguments(args)
					common.FailOnError(err)
				} else if strings.HasPrefix(args, "[map[") {
					cnstrMap[op], err = ParseMapArguments(args)
					common.FailOnError(err)
				} else {
					cnstrMap[op] = args
				}
				//ccList = append(ccList, cnstrMap)
			}
			xmap[mkey] = cnstrMap

			xdata = append(xdata, xmap)
		}
	}
	return xdata
}

func (ntemp *DbPropertyFilter) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {

	compMap, err := dgt.FindNestedComps(uid, key)
	common.FailOnError(err)

	return compMap
}

func (ntemp *DbPropertyFilter) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
