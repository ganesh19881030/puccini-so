package database

import (
	"strings"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/parser"
)

func createScriptNamespace(grammerVersions string) tosca.ScriptNamespace {

	versions := strings.Split(grammerVersions, ",")

	toscaContext := tosca.NewContext(nil, nil)

	for _, ver := range versions {

		grammar := parser.Grammars[ver]
		reader := grammar["MergeScriptNamespace"]

		reader(toscaContext)

	}

	return toscaContext.ScriptNamespace
}
