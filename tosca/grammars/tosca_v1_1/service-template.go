package tosca_v1_1

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/grammars/tosca_v1_3"
)

//
// ServiceTemplate
//
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.9
//

// tosca.Reader signature
func ReadServiceTemplate(context *tosca.Context) interface{} {
	self := tosca_v1_3.NewServiceTemplate(context)
	context.ScriptNamespace.Merge(DefaultScriptNamespace)
	context.ValidateUnsupportedFields(append(context.ReadFields(self), "dsl_definitions"))
	return self
}
