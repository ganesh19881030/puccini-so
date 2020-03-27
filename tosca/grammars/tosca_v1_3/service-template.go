package tosca_v1_3

import (
	"strings"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// ServiceTemplate
//
// See Unit
//
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.10
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.9
//

type ServiceTemplate struct {
	*Unit `name:"service template"`

	Description      *string           `read:"description"`
	TopologyTemplate *TopologyTemplate `read:"topology_template,TopologyTemplate"`
}

func NewServiceTemplate(context *tosca.Context) *ServiceTemplate {
	return &ServiceTemplate{Unit: NewUnit(context)}
}

// tosca.Reader signature
func ReadServiceTemplate(context *tosca.Context) interface{} {
	self := NewServiceTemplate(context)
	context.ScriptNamespace.Merge(DefaultScriptNamespace)
	context.ValidateUnsupportedFields(append(context.ReadFields(self), "dsl_definitions"))
	return self
}

// tosca.Normalizable interface
func (self *ServiceTemplate) Normalize() *normal.ServiceTemplate {
	log.Info("{normalize} service template")

	s := normal.NewServiceTemplate()

	if self.Description != nil {
		s.Description = *self.Description
	}

	s.ScriptNamespace = self.Context.ScriptNamespace

	self.Unit.Normalize(s)
	if self.TopologyTemplate != nil {
		self.TopologyTemplate.Normalize(s)
	}

	return s
}

func AppendTopologyTemplatesInServiceTemplate(currentEntityPtr interface{}, serviceTemplateEntityPtr interface{}, serviceTemplateURLName string) {

	serviceTemplate := serviceTemplateEntityPtr.(*ServiceTemplate).TopologyTemplate
	topologyTemplate := currentEntityPtr.(*ServiceTemplate).TopologyTemplate
	serviceTemplateURLName = strings.ReplaceAll(serviceTemplateURLName, "/", "__")

	// 1.  NodeTemplates
	for _, nodeTemplate := range topologyTemplate.NodeTemplates {

		for _, nodeTemp := range topologyTemplate.NodeTemplates {
			for _, requirement := range nodeTemp.Requirements {
				if requirement.TargetNodeTemplateNameOrTypeName != nil && nodeTemplate.Name == *requirement.TargetNodeTemplateNameOrTypeName {
					*requirement.TargetNodeTemplateNameOrTypeName = serviceTemplateURLName + "." + nodeTemplate.Name
				}
			}
		}

		for _, substitution := range topologyTemplate.SubstitutionMappings {
			for _, requirement := range substitution.RequirementMappings {
				if requirement.NodeTemplateName != nil && nodeTemplate.Name == *requirement.NodeTemplateName {
					*requirement.NodeTemplateName = serviceTemplateURLName + "." + nodeTemplate.Name
				}
			}
			for _, capapability := range substitution.CapabilityMappings {
				if capapability.NodeTemplateName != nil && nodeTemplate.Name == *capapability.NodeTemplateName {
					*capapability.NodeTemplateName = serviceTemplateURLName + "." + nodeTemplate.Name
				}
			}
		}

		for _, capability := range nodeTemplate.Capabilities {
			for _, attribute := range capability.Attributes {
				var modelableEntityName interface{}

				if attribute.Name != "" {
					data := attribute.Context.Data.(*tosca.FunctionCall)
					functionName := data.Name
					modelableEntityName = data.Arguments[0]

					if modelableEntityName != "SELF" && functionName == "get_attribute" {
						data.Arguments[0] = serviceTemplateURLName + "." + modelableEntityName.(string)
					}
				}
			}
		}

		nodeTemplate.Name = serviceTemplateURLName + "." + nodeTemplate.Name
		serviceTemplate.NodeTemplates = append(serviceTemplate.NodeTemplates, nodeTemplate)
	}

	// 2. RelationshipTemplates
	for _, relationshipTemplate := range topologyTemplate.RelationshipTemplates {
		relationshipTemplate.Name = serviceTemplateURLName + "." + relationshipTemplate.Name
		serviceTemplate.RelationshipTemplates = append(serviceTemplate.RelationshipTemplates, relationshipTemplate)
	}

	// 3. Groups
	for _, group := range topologyTemplate.Groups {
		group.Name = serviceTemplateURLName + "." + group.Name
		serviceTemplate.Groups = append(serviceTemplate.Groups, group)
	}

	// 4. Policies
	for _, policy := range topologyTemplate.Policies {
		policy.Name = serviceTemplateURLName + "." + policy.Name
		serviceTemplate.Policies = append(serviceTemplate.Policies, policy)
	}

	// 5. InputParameterDefinitions
	for name, input := range topologyTemplate.InputParameterDefinitions {
		serviceTemplate.InputParameterDefinitions[name] = input
	}
	// 6. OutputParameterDefinitions
	for name, output := range topologyTemplate.OutputParameterDefinitions {

		attribute := output.Value
		var modelableEntityName interface{}

		if attribute.Name != "" {
			data := attribute.Context.Data.(*tosca.FunctionCall)
			functionName := data.Name
			modelableEntityName = data.Arguments[0]

			if modelableEntityName != "SELF" && functionName == "get_attribute" {
				data.Arguments[0] = serviceTemplateURLName + "." + modelableEntityName.(string)
			}
		}

		serviceTemplate.OutputParameterDefinitions[name] = output
	}

	// 7. WorkflowDefinitions
	for name, workflow := range topologyTemplate.WorkflowDefinitions {
		name = serviceTemplateURLName + "." + name
		for _, step := range workflow.StepDefinitions {
			*step.TargetNodeTemplateOrGroupName = serviceTemplateURLName + "." + *step.TargetNodeTemplateOrGroupName
		}
		workflow.Name = name
		serviceTemplate.WorkflowDefinitions[name] = workflow
	}

	// 8. SubstitutionMappings
	for _, substitution := range topologyTemplate.SubstitutionMappings {
		serviceTemplate.SubstitutionMappings = append(serviceTemplate.SubstitutionMappings, substitution)
	}

}

func AppendUnitsInServiceTemplate(currentEntityPtr interface{}, serviceTemplateEntityPtr interface{}, serviceTemplateName string) {

	serviceTemplate := serviceTemplateEntityPtr.(*ServiceTemplate).Unit
	currentUnit := currentEntityPtr.(*ServiceTemplate).Unit

	// 1. Metadata

	// 2. Repositories
	for _, repository := range currentUnit.Repositories {
		serviceTemplate.Repositories = append(serviceTemplate.Repositories, repository)
	}

	// 3.Imports
	for _, imports := range currentUnit.Imports {
		serviceTemplate.Imports = append(serviceTemplate.Imports, imports)
	}

	// 4. ArtifactTypes
	for _, artifactType := range currentUnit.ArtifactTypes {
		serviceTemplate.ArtifactTypes = append(serviceTemplate.ArtifactTypes, artifactType)
	}

	// 5. CapabilityTypes
	for _, capabilityType := range currentUnit.CapabilityTypes {
		serviceTemplate.CapabilityTypes = append(serviceTemplate.CapabilityTypes, capabilityType)
	}

	// 6. DataTypes
	for _, dataType := range currentUnit.DataTypes {
		serviceTemplate.DataTypes = append(serviceTemplate.DataTypes, dataType)
	}

	// 7. GroupTypes
	for _, groupType := range currentUnit.GroupTypes {
		serviceTemplate.GroupTypes = append(serviceTemplate.GroupTypes, groupType)
	}

	// 8. InterfaceTypes
	for _, interfaceType := range currentUnit.InterfaceTypes {
		serviceTemplate.InterfaceTypes = append(serviceTemplate.InterfaceTypes, interfaceType)
	}

	// 9. NodeTypes
	for _, nodeType := range currentUnit.NodeTypes {
		serviceTemplate.NodeTypes = append(serviceTemplate.NodeTypes, nodeType)
	}

	// 10. PolicyTypes
	for _, policyType := range currentUnit.PolicyTypes {
		serviceTemplate.PolicyTypes = append(serviceTemplate.PolicyTypes, policyType)
	}

	// 11. RelationshipTypes
	for _, relationshipType := range currentUnit.RelationshipTypes {
		serviceTemplate.RelationshipTypes = append(serviceTemplate.RelationshipTypes, relationshipType)
	}
}
