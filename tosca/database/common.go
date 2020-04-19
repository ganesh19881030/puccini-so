package database

import (
	"reflect"
	"strings"

	"github.com/op/go-logging"
)

var Log = logging.MustGetLogger("database")

// ExtractTopologyName
func ExtractTopologyName(urlString string) string {

	ind := strings.LastIndex(urlString, "/")

	var topologyName = urlString
	if ind == -1 {
		ind = strings.LastIndex(urlString, "\\")
	}
	if ind > -1 {
		topologyName = urlString[ind+1:]
	}
	ind = strings.LastIndex(topologyName, ".")
	if ind > -1 {
		topologyName = topologyName[:ind]
	}

	return topologyName
}

//GetEntityType - fetches the entity type string which should match its type in Dgraph
func GetEntityType(entityPtr interface{}) string {
	strType := reflect.ValueOf(entityPtr).Type().String()
	parts := strings.Split(strType, ".")
	strType = parts[1]
	return strType
}
