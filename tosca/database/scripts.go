package database

import (
	//"fmt"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca/problems"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/parser"
	"github.com/tliron/puccini/url"
)

// CreateScriptNamespace - creates java script namespace
func CreateScriptNamespace(grammerVersions string, internalImport string) tosca.ScriptNamespace {

	versions := strings.Split(grammerVersions, ",")

	problems := new(problems.Problems)

	toscaContext := tosca.NewContext(problems, nil)

	for _, ver := range versions {
		var grammar tosca.Grammar
		if toscaDef := parser.Grammars["tosca_definitions_version"]; toscaDef != nil {
			grammar = toscaDef[ver]
		} else if toscaDef := parser.Grammars["heat_template_version"]; toscaDef != nil {
			grammar = toscaDef[ver]
		}

		reader := grammar["MergeScriptNamespace"]

		reader(toscaContext)
		paths := make([]string, 0)

		toscaDef := parser.InternalProfilePaths["tosca_definitions_version"]
		toscaPath := toscaDef[ver]
		//kubePath := "/tosca/kubernetes/1.0/profile.yaml"
		paths = append(paths, toscaPath)
		paths = append(paths, internalImport)

		for _, path := range paths {
			if profileURL, err := url.NewValidInternalURL(path); err == nil {
				data, _, _ := ard.ReadURL(profileURL, true)
				toscaContext.URL = profileURL
				toscaContext.Data = data["metadata"]
				reader := grammar["Metadata"]
				reader(toscaContext)
			}

		}

	}

	return toscaContext.ScriptNamespace
}
