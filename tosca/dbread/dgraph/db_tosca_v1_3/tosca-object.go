package db_tosca_v1_3

import (
	"errors"
	"fmt"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbToscaObject struct{}

func (ntemp *DbToscaObject) Convert(responseData *interface{}, readerName string) interface{} {
	return nil
}

func (ntemp *DbToscaObject) ConvertMap(responseData *[]interface{}, key string, readername string) interface{} {
	return nil
}

func (ntemp *DbToscaObject) ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{} {

	return nil

}

func (ntemp *DbToscaObject) DbRead(dgt *dgraph.DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map {
	return nil
}

func (ntemp DbToscaObject) ByPassDbRead(contextData *interface{}, name string, key string) bool {
	return false
}

func (ntemp *DbToscaObject) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, string, error) {
	obj, ok := searchObject.(dgraph.SearchFields)
	if ok {
		fnd, uid, err := dgt.FindComp(obj.ObjectKey, obj.ObjectDGType, obj.ObjectNSuid, obj.SubjectUid, obj.Predicate)
		return fnd, uid, obj.ObjectKey, err
	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DbFind function."))
	}
	return false, "", obj.ObjectKey, nil
}

func (ntemp *DbToscaObject) DbInsert(dgt *dgraph.DgraphTemplate, mutateQuery string) (string, error) {
	fmt.Println("nquad: ", mutateQuery)

	resp, err := dgt.ExecMutation(mutateQuery)

	uid := ""
	if err != nil {
		return uid, err
	} else {
		fmt.Println("Assigned UUIDs : ")
		for key, value := range resp.Uids {
			uid = value
			fmt.Println("name:", key, ",  value:", value)
		}
		fmt.Println("json: ", string(resp.GetJson()))
		uid = resp.Uids["comp"]
	}

	return uid, err

}

func (ntemp *DbToscaObject) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	return "", nil
}
