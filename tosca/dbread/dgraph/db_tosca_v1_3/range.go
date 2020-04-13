package db_tosca_v1_3

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type RangeQuery struct{}

func (rq RangeQuery) Query(field reflect.Value) string {
	vrange := field.Interface()
	value := reflect.ValueOf(vrange)
	num := value.NumField()
	fmt.Printf("\nnum of range fields: %d\n", num)
	var nquad string
	for ind := 0; ind < num; ind++ {
		field := value.Field(ind)
		//fieldType := field.Type()
		nam := value.Type().Field(ind).Name
		f := field.Interface()
		val := reflect.ValueOf(f)
		triple := fmt.Sprintf(`
	_:comp <%s> "%d" .`, strings.ToLower(nam), val)
		nquad = nquad + triple

	}
	return nquad
}

func init() {
	dgraph.FieldRegistries.QueryRegistry["Range"] = new(RangeQuery)
}

type DbRange struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbRange) ByPassDbRead(contextData *interface{}, name string, key string) bool {
	childData := (*contextData).(ard.Map)[key]
	lo := (childData.(ard.Map)["lower"]).(string)
	up := (childData.(ard.Map)["upper"]).(string)
	var rng ard.List
	loi, err := strconv.ParseInt(lo, 10, 32)
	common.FailOnError(err)
	rng = append(rng, loi)
	upi, err := strconv.ParseInt(up, 10, 64)
	if err != nil {
		rng = append(rng, "UNBOUNDED")
	} else {
		rng = append(rng, upi)
	}
	(*contextData).(ard.Map)[key] = rng

	return true
}

func (ntemp *DbRange) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := `_:comp <dgraph.type> "%s" .`
		nquad = fmt.Sprintf(nquad, saveFields.DgType)
		nquad = dgraph.AddFields(saveFields.EntityPtr, nquad, saveFields.DgType)

		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
