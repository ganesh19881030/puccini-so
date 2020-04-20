package db_tosca_v1_3

import (
	"errors"
	"strconv"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

type DbOccurrences struct {
	DbToscaObject // embedded structure that has the basic functionality
}

func (ntemp *DbOccurrences) ByPassDbRead(contextData *interface{}, name string, key string) bool {
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

func (ntemp *DbOccurrences) DbBuildInsertQuery(dataObject interface{}) (string, error) {
	if saveFields, ok := dataObject.(dgraph.SaveFields); ok {
		nquad := dgraph.BuildInsertQuery(saveFields)
		return nquad, nil
	} else {
		return "", errors.New(ErrorInvalidSaveDataObject)
	}
}
