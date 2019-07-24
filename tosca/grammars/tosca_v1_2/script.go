package tosca_v1_2

import (
	"github.com/tliron/puccini/tosca"
)

// MergeScriptNamespace to merge scripts
// tosca.Reader signature
func MergeScriptNamespace(context *tosca.Context) interface{} {
	context.ScriptNamespace.Merge(DefaultScriptNamespace)

	return nil
}
