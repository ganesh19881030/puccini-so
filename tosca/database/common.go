package database

import (
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
