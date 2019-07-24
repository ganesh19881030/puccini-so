package database

import (
	"strings"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/parser"
)

// CreateScriptNamespace - creates java script namespace
func CreateScriptNamespace(grammerVersions string) tosca.ScriptNamespace {

	versions := strings.Split(grammerVersions, ",")

	toscaContext := tosca.NewContext(nil, nil)

	for _, ver := range versions {

		grammar := parser.Grammars[ver]
		reader := grammar["MergeScriptNamespace"]

		reader(toscaContext)

	}

	return toscaContext.ScriptNamespace
}
