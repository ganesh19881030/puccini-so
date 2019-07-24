package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
)

// MergeScriptNamespaceUnit to merge scripts
// tosca.Reader signature
func MergeScriptNamespace(context *tosca.Context) interface{} {
	context.ScriptNamespace.Merge(DefaultScriptNamespace)

	return nil
}
