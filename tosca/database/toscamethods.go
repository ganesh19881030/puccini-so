/*
 This package contains methods that are used by objects in dbread/dgraph package.
 Any reference to tosca package in dbread/... packages creates an import cycle which
 gives compilation errors.

 Therefore, here we register methods that reference tosca package objects/functions
 in a registry, for later use by objects in  dbread and underlying packages.

 This probably is a hack that should be resolved by a better design of the database
 persistence objects
*/
package database

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

// QueryValue object type to contain the Tosca method for Value objects
type QueryValue struct{}

// Process - implementation of the tosca method for Values to add to the
//           insert query
func (av QueryValue) Process(entityPtr *interface{}, nam string) string {
	var nquad, val string
	ctxt := tosca.GetContext(*entityPtr)
	switch ctxt.Data.(type) {
	case string:
		val = fmt.Sprintf("%s", ctxt.Data)
		nquad = nquad + `
	_:comp <myvaluetype> "string" .`
	case int:
		val = fmt.Sprintf("%v", ctxt.Data)
		nquad = nquad + `
	_:comp <myvaluetype> "int" .`
	case bool:
		val = fmt.Sprintf("%v", ctxt.Data)
		nquad = nquad + `
	_:comp <myvaluetype> "bool" .`
	case float64:
		val = fmt.Sprintf("%v", ctxt.Data)
		nquad = nquad + `
	_:comp <myvaluetype> "float" .`
	}
	//val, ok := ctxt.Data.(string)
	if val != "" {
		nquad = nquad + fmt.Sprintf(`
	_:comp <myvalue> "%s" .`, val)
	} else {
		//if bag.Predicate == "attributes" {
		xfunc, ok := ctxt.Data.(*tosca.FunctionCall)
		if ok {
			nquad = nquad + fmt.Sprintf(`
_:comp <functionname> "%s" .`, xfunc.Name)
			strargs := "["
			for ind, arg := range xfunc.Arguments {
				if ind > 0 {
					strargs = strargs + " "
				}
				strargs = strargs + fmt.Sprintf("%s", arg)
			}
			strargs = strargs + "]"
			nquad = nquad + fmt.Sprintf(`
_:comp <fnarguments> "%s" .`, strargs)

		}
	}

	return nquad

}

// QueryAttributeDefault object type to contain the Tosca method for default values
type QueryAttributeDefault struct{}

// Process - implementation of the tosca method for attribute default values to add to the
//           insert query
func (av QueryAttributeDefault) Process(entityPtr *interface{}, nam string) string {
	var nquad string
	var ok bool
	var adata ard.Map
	var dflt interface{}

	ctx := tosca.GetContext(*entityPtr)
	adata, ok = ctx.Data.(ard.Map)

	if ok {
		dflt, ok = adata["default"]
		if ok {
			val := reflect.ValueOf(dflt)
			if val.IsValid() {
				triple := fmt.Sprintf(`
	_:comp <%s> "%v" .`, strings.ToLower(nam), val)
				nquad = nquad + triple
			}
		}
	}

	return nquad

}

func init() {
	// register the tosca methods with the ToscaMethodRegistry in dgraph package
	dgraph.ToscaMethodRegistry[dgraph.QueryValueKey] = new(QueryValue)
	dgraph.ToscaMethodRegistry[dgraph.QueryAttributeDefaultKey] = new(QueryAttributeDefault)

}
