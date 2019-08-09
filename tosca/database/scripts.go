package database

import (
	//"fmt"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca/problems"
	"strings"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/parser"
	"github.com/tliron/puccini/url"
)

func CreateScriptNamespace(grammerVersions string, internalImport string) tosca.ScriptNamespace {
	versions := strings.Split(grammerVersions, ",")

	problems := new(problems.Problems)

	toscaContext := tosca.NewContext(problems, nil)

	for _, ver := range versions {

		grammar := parser.Grammars[ver]
		reader := grammar["MergeScriptNamespace"]

		reader(toscaContext)
		paths := make([]string, 0)

		toscaPath := parser.ProfileInternalPaths[ver]
		//kubePath := "/tosca/kubernetes/1.0/profile.yaml"
		paths = append(paths, toscaPath)
		paths = append(paths, internalImport)

		for _, path := range paths {
			if profileURL, err := url.NewValidInternalURL(path); err == nil {
				data, _ := ard.ReadURL(profileURL)
				toscaContext.URL = profileURL
				toscaContext.Data = data["metadata"]
				reader := grammar["Metadata"]
				reader(toscaContext)
			}

		}

	}

	return toscaContext.ScriptNamespace
}
