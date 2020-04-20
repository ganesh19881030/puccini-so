package db_tosca_v1_3

import (
	"errors"
	"fmt"

	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbDGNamespace struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbDGNamespace) DbFind(dgt *dgraph.DgraphTemplate, searchObject interface{}) (bool, string, error) {
	srchFlds, ok := searchObject.(dgraph.SearchFields)
	if ok {
		return dgt.FindNamespace(srchFlds.ObjectKey)
	} else {
		common.FailOnError(errors.New("Invalid searchObject passed to DGNamespace.DbFind function."))
	}
	return false, "", nil
}
func (ntemp *DbDGNamespace) DbInsert(dgt *dgraph.DgraphTemplate, mutateQuery string) (string, error) {

	var uid string
	log.Debugf("Namespace insert query : %s", mutateQuery)

	resp, err := dgt.ExecMutation(mutateQuery)

	if err == nil {
		log.Debugf("Assigned UUIDs : ")
		for key, value := range resp.Uids {
			uid = value
			log.Debugf("name: %s  value: %s", key, value)
		}
		log.Debugf("json: %s", string(resp.GetJson()))
	}

	return uid, err

}

func (ntemp *DbDGNamespace) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	nsFields, ok := dataObject.(dgraph.Namespace)
	if ok {
		nquad := `_:namespace <dgraph.type> "Namespace" .
		_:namespace <url> "%s" .
		_:namespace <version> "%s" .`
		nquad = fmt.Sprintf(nquad, nsFields.Url, nsFields.Version)

		return nquad, nil

	} else {
		common.FailOnError(errors.New("Invalid search fields Object passed to DGNamespace.DbBuildInsertQuery function."))
	}

	return "", nil
}
