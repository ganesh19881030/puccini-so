package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/op/go-logging"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/tosca/database"
	"github.com/tliron/puccini/url"
)

var log = logging.MustGetLogger("so")

var output string

func ReadClout(path string) (*clout.Clout, error) {
	var url_ url.URL

	var err error
	if path != "" {
		url_, err = url.NewValidURL(path, nil)
	} else {
		url_, err = url.ReadInternalURLFromStdin("yaml")
	}
	if err != nil {
		return nil, err
	}

	reader, err := url_.Open()
	if err != nil {
		return nil, err
	}

	if readerCloser, ok := reader.(io.ReadCloser); ok {
		defer readerCloser.Close()
	}

	f := url_.Format()

	switch f {
	case "json":
		return clout.DecodeJson(reader)
	case "yaml", "":
		return clout.DecodeYaml(reader)
	case "xml":
		return clout.DecodeXml(reader)
	default:
		return nil, fmt.Errorf("unsupported format: %s", f)
	}
}

// ReadCloutFromDgraph reads the clout data from Dgraph
func ReadCloutFromDgraph(name string) (*clout.Clout, string, error) {
	// construct Dgraph url from configuration
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)

	//f := url_.Format()
	output, uid, err := createCloutOutput(dburl, name)

	return output, uid, err

}

// CloutInstanceExists checks if a clout instance exists in database
func CloutInstanceExists(name string) bool {
	// construct Dgraph url from configuration]
	dgt, err := fetchDbTemplate()
	common.FailOnError(err)
	defer dgt.Close()
	tpname := database.ExtractTopologyName(name)
	found, _ := isCloutPresent(dgt, tpname)

	return found

}
func ParseInputsFromUrl(inputsUrl string) ard.Map {
	inputValues := make(ard.Map)
	if inputsUrl != "" {
		//log.Infof("load inputs from %s", inputsUrl)
		url_, err := url.NewValidURL(inputsUrl, nil)
		common.FailOnError(err)
		reader, err := url_.Open()
		common.FailOnError(err)
		if readerCloser, ok := reader.(io.ReadCloser); ok {
			defer readerCloser.Close()
		}
		data, err := format.Read(reader, "yaml")
		common.FailOnError(err)
		if map_, ok := data.(ard.Map); ok {
			for key, value := range map_ {
				inputValues[key] = value
			}
		} else {
			common.Failf("malformed inputs in %s", inputsUrl)
		}
	}
	return inputValues

}
func ParseInputsFromCommandLine(inputs []string) ard.Map {
	inputValues := make(ard.Map)
	for _, input := range inputs {
		s := strings.SplitN(input, "=", 2)
		if len(s) != 2 {
			common.Failf("malformed input: %s", input)
		}
		value, err := format.Decode(s[1], "yaml")
		common.FailOnError(err)
		inputValues[s[0]] = value
	}
	return inputValues
}
