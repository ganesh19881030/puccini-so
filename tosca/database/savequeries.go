package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/dbread"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

var dbsave bool
var dbObjectMap dgraph.DbObjectMap
var nsObject dgraph.Persistable

func init() {
	//dbsave = false
	dbsave = true
	dbObjectMap = dbread.DbGrammar["dgraph.tosca_v1_3"]
	var ok bool
	if nsObject, ok = dbObjectMap["DGNamespace"]; !ok {
		common.FailOnError(errors.New("No object defined for DGNamespace"))
	}
}

func PersistNamespace(urls string, version string, urlmap map[string]string, dgt *dgraph.DgraphTemplate) {
	var err error
	uid := urlmap[urls]
	var fnd bool
	if !dbsave {
		return
	}
	if uid == "" {
		so := dgraph.SearchFields{
			ObjectKey: urls,
		}
		//fnd, uid, err = findNamespace(urls, dgt)
		fnd, uid, err = nsObject.DbFind(dgt, so)
		common.FailOnError(err)
		if !fnd {
			//uid, err = insertNamespace(urls, version, dgt)
			ns := dgraph.Namespace{
				Url:     urls,
				Version: version,
			}
			var mquery string
			mquery, err = nsObject.DbBuildInsertQuery(ns)
			common.FailOnError(err)
			uid, err = nsObject.DbInsert(dgt, mquery)
			common.FailOnError(err)
		}
		urlmap[urls] = uid

	}
}

func PersistToscaComponent(entityPtr interface{}, name string, dgtype string, nurl string, urlmap map[string]string, bag *TravelBag) (string, error) {

	var err error
	var fnd bool
	var uid string
	var dbObject dgraph.Persistable
	var ok bool
	if !dbsave {
		return uid, nil
	}

	if dbObject, ok = dbObjectMap[dgtype]; !ok {
		common.FailOnError(errors.New("DB object not defined for [" + dgtype + "]"))
	}

	nsuid := urlmap[nurl]

	so := dgraph.SearchFields{
		ObjectKey:    name,
		ObjectDGType: dgtype,
		ObjectNSuid:  nsuid,
		SubjectUid:   bag.Uid,
		Predicate:    bag.Predicate,
	}
	fnd, uid, err = dbObject.DbFind(bag.Dgt, so)
	if err == nil {
		if !fnd {
			saveObj := dgraph.SaveFields{
				EntityPtr:  entityPtr,
				Name:       name,
				DgType:     dgtype,
				Nurl:       nurl,
				Nsuid:      nsuid,
				SubjectUid: bag.Uid,
				Predicate:  bag.Predicate,
			}
			var nquad string
			nquad, err = dbObject.DbBuildInsertQuery(saveObj)
			if err == nil {
				uid, err = dbObject.DbInsert(bag.Dgt, nquad)
			}
		}
	}
	return uid, err
}

func linkExists(uidObject string, bag *TravelBag) (bool, error) {
	fnd := false
	paramQuery := `
	query {
		comp(func: uid(<%s>)) @cascade {
			uid
			%s @filter(uid(<%s>)){
			  uid
			}
		}
	}`

	notfoundstr := `"comp":[]`

	queryStmt := fmt.Sprintf(paramQuery, bag.Uid, bag.Predicate, uidObject)

	if bag.Uid == uidObject {
		return false, errors.New(fmt.Sprintf("Cannot find link object [%s] to same object [%s]", bag.Uid, uidObject))
	}

	Log.Debugf("Check Link exists query: %s", queryStmt)

	resp, err := bag.Dgt.ExecQuery(queryStmt)

	if err == nil {
		bytes := resp.GetJson()
		respstr := string(bytes)
		fnd = !strings.Contains(respstr, notfoundstr)
	}

	return fnd, err
}

func LinkToscaComponents(uidObject string, bag *TravelBag) (bool, error) {

	if !dbsave {
		return false, nil
	}
	nquad := `<%s> <%s> <%s> .`

	nquad = fmt.Sprintf(nquad, bag.Uid, bag.Predicate, uidObject)

	if bag.Uid == uidObject {
		return false, errors.New(fmt.Sprintf("Cannot link object [%s] to same object [%s]", bag.Uid, uidObject))
	}
	_, ok := linkmap[nquad]
	if ok {
		return true, nil
	} else {
		linkmap[nquad] = true
	}

	lexst, err := linkExists(uidObject, bag)

	if !lexst {
		Log.Debugf("Link TOSCA comps nquad: %s", nquad)

		_, err = bag.Dgt.ExecMutation(nquad)
	}

	return false, err
}

func AddFieldToComponent(entityPtr interface{}, bag *TravelBag) error {

	if !dbsave {
		return nil
	}

	nquad := `<%s> <%s> "%s" .`

	ctxt := tosca.GetContext(entityPtr)
	valstr := fmt.Sprintf("%v", reflect.ValueOf(ctxt.Data))
	valstr2, err := getPredicateValue(bag.Predicate, bag)
	if err == nil && valstr == valstr2 {
		return nil
	}

	nquad = fmt.Sprintf(nquad, bag.Uid, bag.Predicate, valstr)

	Log.Debugf("Add field to TOSCA comp nquad: %s", nquad)

	_, err = bag.Dgt.ExecMutation(nquad)

	return err
}
func getPredicateValue(predicate string, bag *TravelBag) (string, error) {
	var val string
	var err error

	if !dbsave {
		return val, nil
	}

	queryTemplate := `
	query {
		comp(func: has(%s)) @filter(uid(<%s>)){
			%s
		}
	}`

	queryStmt := fmt.Sprintf(queryTemplate, predicate, bag.Uid, predicate)
	resp, err := bag.Dgt.ExecQuery(queryStmt)

	if err == nil {

		jsonstr := string(resp.GetJson())
		Log.Debugf("json: %s", jsonstr)
		ind := strings.Index(jsonstr, predicate)
		if ind > -1 {
			ind1 := strings.Index(jsonstr[ind:], ":")
			if ind1 > -1 {
				ind2 := strings.Index(jsonstr[ind+ind1:], "\"")
				if ind2 > -1 {
					ind3 := strings.Index(jsonstr[ind+ind1+ind2+1:], "\"")
					if ind3 > -1 {
						val = jsonstr[ind+ind1+ind2+1 : ind+ind1+ind2+ind3+1]
					}

				}
			}
		}

	}
	return val, err
}
