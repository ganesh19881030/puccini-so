package db_tosca_v1_3

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type MetadataQuery struct{}

func (mq MetadataQuery) Query(field reflect.Value) string {
	var nquad string
	nquad = nquad + `
	_:mdata <dgraph.type> "Metadata" .`
	iter := field.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		triple := fmt.Sprintf(`
	_:mdata <%s> "%s" .`, k, v)
		nquad = nquad + triple
	}
	nquad = nquad + `
	_:comp <metadata> _:mdata .`

	return nquad
}

func init() {
	dgraph.FieldRegistries.QueryRegistry["Metadata"] = new(MetadataQuery)
}

type DbMetadata struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbMetadata) ByPassDbRead(contextData *interface{}, name string, key string) bool {
	childData := (*contextData).(ard.Map)[key]
	norm, ok := childData.(ard.Map)["normative"]
	if ok {
		if reflect.ValueOf(norm).Type().Kind() == reflect.Bool {
			norms := strconv.FormatBool(norm.(bool))
			childData.(ard.Map)["normative"] = norms
			(*contextData).(ard.Map)[key] = childData
		}
	}

	return true
}
func (ntemp *DbMetadata) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
