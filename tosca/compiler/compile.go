package compiler

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/tosca/normal"
)

func Compile(s *normal.ServiceTemplate) (*clout.Clout, error) {
	clout_ := clout.NewClout()

	timestamp, err := common.Timestamp()
	if err != nil {
		return nil, err
	}

	metadata := make(ard.Map)
	for name, jsEntry := range s.ScriptNamespace {
		sourceCode, err := jsEntry.GetSourceCode()
		if err != nil {
			return nil, err
		}
		if err = js.SetMapNested(metadata, name, sourceCode); err != nil {
			return nil, err
		}
	}
	clout_.Metadata["puccini-js"] = metadata

	metadata = make(ard.Map)
	metadata["version"] = VERSION
	metadata["history"] = []string{timestamp}
	clout_.Metadata["puccini-tosca"] = metadata

	tosca := make(ard.Map)
	tosca["description"] = s.Description
	tosca["metadata"] = s.Metadata
	tosca["inputs"] = s.Inputs
	tosca["outputs"] = s.Outputs
	clout_.Properties["tosca"] = tosca

	nodeTemplates := make(map[string]*clout.Vertex)

	// Node templates
	for _, nodeTemplate := range s.NodeTemplates {
		v := clout_.NewVertex(clout.NewKey())

		nodeTemplates[nodeTemplate.Name] = v

		SetMetadata(v, "nodeTemplate")
		v.Properties["name"] = nodeTemplate.Name
		v.Properties["description"] = nodeTemplate.Description
		v.Properties["types"] = nodeTemplate.Types
		v.Properties["directives"] = nodeTemplate.Directives
		v.Properties["properties"] = nodeTemplate.Properties
		v.Properties["attributes"] = nodeTemplate.Attributes
		v.Properties["requirements"] = nodeTemplate.Requirements
		v.Properties["capabilities"] = nodeTemplate.Capabilities
		v.Properties["interfaces"] = nodeTemplate.Interfaces
		v.Properties["artifacts"] = nodeTemplate.Artifacts
	}

	groups := make(map[string]*clout.Vertex)

	// Groups
	for _, group := range s.Groups {
		v := clout_.NewVertex(clout.NewKey())

		groups[group.Name] = v

		SetMetadata(v, "group")
		v.Properties["name"] = group.Name
		v.Properties["description"] = group.Description
		v.Properties["types"] = group.Types
		v.Properties["properties"] = group.Properties
		v.Properties["interfaces"] = group.Interfaces

		for _, nodeTemplate := range group.Members {
			nv := nodeTemplates[nodeTemplate.Name]
			e := v.NewEdgeTo(nv)

			SetMetadata(e, "member")
		}
	}

	workflows := make(map[string]*clout.Vertex)

	// Workflows
	for _, workflow := range s.Workflows {
		v := clout_.NewVertex(clout.NewKey())

		workflows[workflow.Name] = v

		SetMetadata(v, "workflow")
		v.Properties["name"] = workflow.Name
		v.Properties["description"] = workflow.Description
	}

	// Workflow steps
	for name, workflow := range s.Workflows {
		v := workflows[name]

		steps := make(map[string]*clout.Vertex)

		for _, step := range workflow.Steps {
			sv := clout_.NewVertex(clout.NewKey())

			steps[step.Name] = sv

			SetMetadata(sv, "workflowStep")
			sv.Properties["name"] = step.Name

			e := v.NewEdgeTo(sv)
			SetMetadata(e, "workflowStep")

			if step.TargetNodeTemplate != nil {
				nv := nodeTemplates[step.TargetNodeTemplate.Name]
				e = sv.NewEdgeTo(nv)
				SetMetadata(e, "nodeTemplateTarget")
			} else if step.TargetGroup != nil {
				gv := groups[step.TargetGroup.Name]
				e = sv.NewEdgeTo(gv)
				SetMetadata(e, "groupTarget")
			} else {
				// This would happen only if there was a parsing error
				continue
			}

			// Workflow activities
			for sequence, activity := range step.Activities {
				av := clout_.NewVertex(clout.NewKey())

				e = sv.NewEdgeTo(av)
				SetMetadata(e, "workflowActivity")
				e.Properties["sequence"] = sequence

				SetMetadata(av, "workflowActivity")
				if activity.DelegateWorkflow != nil {
					wv := workflows[activity.DelegateWorkflow.Name]
					e = av.NewEdgeTo(wv)
					SetMetadata(e, "delegateWorflow")
				} else if activity.InlineWorkflow != nil {
					wv := workflows[activity.InlineWorkflow.Name]
					e = av.NewEdgeTo(wv)
					SetMetadata(e, "inlineWorflow")
				} else if activity.SetNodeState != "" {
					av.Properties["setNodeState"] = activity.SetNodeState
				} else if activity.CallOperation != nil {
					m := make(ard.Map)
					m["interface"] = activity.CallOperation.Interface.Name
					m["operation"] = activity.CallOperation.Name
					av.Properties["callOperation"] = m
				}
			}
		}

		for _, step := range workflow.Steps {
			sv := steps[step.Name]

			for _, next := range step.OnSuccessSteps {
				nsv := steps[next.Name]
				e := sv.NewEdgeTo(nsv)
				SetMetadata(e, "onSuccess")
			}

			for _, next := range step.OnFailureSteps {
				nsv := steps[next.Name]
				e := sv.NewEdgeTo(nsv)
				SetMetadata(e, "onFailure")
			}
		}
	}

	// Policies
	for _, policy := range s.Policies {
		v := clout_.NewVertex(clout.NewKey())

		SetMetadata(v, "policy")
		v.Properties["name"] = policy.Name
		v.Properties["description"] = policy.Description
		v.Properties["types"] = policy.Types
		v.Properties["properties"] = policy.Properties

		for _, nodeTemplate := range policy.NodeTemplateTargets {
			nv := nodeTemplates[nodeTemplate.Name]
			e := v.NewEdgeTo(nv)

			SetMetadata(e, "nodeTemplateTarget")
		}

		for _, group := range policy.GroupTargets {
			gv := groups[group.Name]
			e := v.NewEdgeTo(gv)

			SetMetadata(e, "groupTarget")
		}

		for _, trigger := range policy.Triggers {
			if trigger.Operation != nil {
				to := clout_.NewVertex(clout.NewKey())

				SetMetadata(to, "operation")
				to.Properties["description"] = trigger.Operation.Description
				to.Properties["implementation"] = trigger.Operation.Implementation
				to.Properties["dependencies"] = trigger.Operation.Dependencies
				to.Properties["inputs"] = trigger.Operation.Inputs

				e := v.NewEdgeTo(to)
				SetMetadata(e, "policyTriggerOperation")
			} else if trigger.Workflow != nil {
				wv := workflows[trigger.Workflow.Name]

				e := v.NewEdgeTo(wv)
				SetMetadata(e, "policyTriggerWorkflow")
			}
			if trigger.Condition != nil {
				conditionClauses := make(map[string]normal.FunctionCalls)
				tc := clout_.NewVertex(clout.NewKey())
				SetMetadata(tc, "condition")

				for name, conditionClause := range trigger.Condition.ConditionClauseConstraints {
					conditionClauses[name] = conditionClause
				}
				tc.Properties["conditionClauses"] = conditionClauses

				e := v.NewEdgeTo(tc)
				SetMetadata(e, "policyTriggerCondition")
			}
			if trigger.Action != nil {
				ta := clout_.NewVertex(clout.NewKey())
				SetMetadata(ta, "action")

				if trigger.Action.Update != nil {
					ta.Properties["update"] = trigger.Action.Update
				}

				e := v.NewEdgeTo(ta)
				SetMetadata(e, "policyTriggerAction")
			}
		}
	}

	// Substitution
	for _, substitution := range s.Substitution {
		if substitution != nil {
			v := clout_.NewVertex(clout.NewKey())

			SetMetadata(v, "substitution")
			v.Properties["type"] = substitution.Type
			v.Properties["typeMetadata"] = substitution.TypeMetadata
			v.Properties["properties"] = substitution.PropertyMappings
			v.Properties["substitutionFilter"] = substitution.SubstitutionFilters

			for nodeTemplate, capability := range substitution.CapabilityMappings {
				vv := nodeTemplates[nodeTemplate.Name]
				e := v.NewEdgeTo(vv)

				SetMetadata(e, "capabilityMapping")
				e.Properties["capability"] = capability.Name
			}

			for requirementName, requirement := range substitution.RequirementMappings {

				n := requirement.SourceNodeTemplate
				vv := nodeTemplates[n.Name]
				e := v.NewEdgeTo(vv)

				SetMetadata(e, "requirementMapping")
				e.Properties["requirement"] = requirement.Name
				e.Properties["requirementName"] = requirementName
			}

			for nodeTemplate, interface_ := range substitution.InterfaceMappings {
				vv := nodeTemplates[nodeTemplate.Name]
				e := v.NewEdgeTo(vv)

				SetMetadata(e, "interfaceMapping")
				e.Properties["interface"] = interface_
			}
		}
	}

	// Normalize
	clout_, err = clout_.Normalize()
	if err != nil {
		return clout_, err
	}

	// code added to handle substitution mappings

	// when csar/zip is passed to compiler, make connection of abstract
	// node template and implementation nodes by adding vertexIDs of
	// implementation nodes to the substitute directives of abstract node template
	cloutVertexes := clout_.Vertexes

	// look for abstract nodes in the clout
	for _, abstractVertex := range cloutVertexes {

		// ignore vertexes other than "node-template"
		if !isVertexOfSpecificKind(abstractVertex, "nodeTemplate") {
			continue
		}

		// if directives is empty, its not an abstract node
		vertexProperties := abstractVertex.Properties
		directiveList := vertexProperties["directives"].([]interface{})
		length := len(directiveList)
		if length == 0 {
			continue
		}

		// abstract node found, look for its substitute directive
		var newDirectives []string
		for _, directive := range directiveList {
			if directive == "substitute" {

				// look for matching node type between abstract and substitute
				sourceVertexTypes := vertexProperties["types"].(map[string]interface{})
				for sourceVertexType := range sourceVertexTypes {
					for _, substituteVertex := range cloutVertexes {

						// ignore vertexes other than "substitution"
						if !isVertexOfSpecificKind(substituteVertex, "substitution") {
							continue
						}
						vertexPropeties := substituteVertex.Properties
						vertexType := vertexPropeties["type"].(string)
						if sourceVertexType == vertexType {
							if !checkForSubstitutionFilter(abstractVertex, substituteVertex, clout_) {
								continue
							}
							vertexesOfSpecificServiceTemplate := make(clout.Vertexes)
							newDirectiveName := directive.(string) + ":" + substituteVertex.ID

							// substitute vertex found, using its edgesOut, find its implementation node templates
							for _, edge := range substituteVertex.EdgesOut {
								edgeMap := edge.GetMetadata()
								ptosca, _ := edgeMap["puccini-tosca"].(map[string]interface{})
								kind, _ := ptosca["kind"]
								if kind != "requirementMapping" && kind != "capabilityMapping" {
									continue
								}

								vertexOfServiceTemplate := findVertexBasedOnID(edge.TargetID, cloutVertexes)
								vertexesOfSpecificServiceTemplate[vertexOfServiceTemplate.ID] = vertexOfServiceTemplate

								// collect node template IDs of substitute template's requirements
								requirements, _ := vertexOfServiceTemplate.Properties["requirements"].([]interface{})
								for _, requirement := range requirements {
									reqMap, _ := requirement.(map[string]interface{})
									nodeTemplateName, _ := reqMap["nodeTemplateName"]
									if nodeTemplateName != "" {
										for _, vertexFromClout := range cloutVertexes {
											vertexPropertiesMap := vertexFromClout.Properties
											vertexName, _ := vertexPropertiesMap["name"]
											if (vertexName != nil) && (vertexName == nodeTemplateName) {
												vertexesOfSpecificServiceTemplate[vertexFromClout.ID] = vertexFromClout
											}
										}
									}
								}

							}

							// look for requirements of node templates found till now
							for _, vertex := range vertexesOfSpecificServiceTemplate {
								vertexProperties := vertex.Properties
								vertexName := vertexProperties["name"]

								for _, vertexFromClout := range cloutVertexes {
									vertexFromCloutProperties := vertexFromClout.Properties
									requirements, _ := vertexFromCloutProperties["requirements"].([]interface{})
									for _, requirement := range requirements {
										reqMap, _ := requirement.(map[string]interface{})
										nodeTemplateName := reqMap["nodeTemplateName"]
										if nodeTemplateName == vertexName {
											vertexesOfSpecificServiceTemplate[vertexFromClout.ID] = vertexFromClout
										}
									}
								}
							}
							for _, vertexeOfSpecificServiceTemplate := range vertexesOfSpecificServiceTemplate {
								newDirectiveName = newDirectiveName + ":" + vertexeOfSpecificServiceTemplate.ID
							}
							newDirectives = append(newDirectives, newDirectiveName)
						}
					}
				}
			} else {
				newDirectives = append(newDirectives, directive.(string))
			}
		}
		if len(newDirectives) != 0 {
			abstractVertex.Properties["directives"] = newDirectives
		}

	}

	// TODO: call JavaScript plugins

	return clout_, nil
}

func SetMetadata(entity clout.Entity, kind string) {
	metadata := make(ard.Map)
	metadata["version"] = VERSION
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

func isVertexOfSpecificKind(vertex *clout.Vertex, vertexKind string) bool {
	isVertexKindMatch := false
	vertexMetadata := vertex.Metadata
	if vertexMetadata != nil {
		metaDataName := vertexMetadata["puccini-tosca"].(map[string]interface{})
		kindName := metaDataName["kind"].(interface{})
		if kindName.(string) == vertexKind {
			isVertexKindMatch = true
		}
	}
	return isVertexKindMatch
}

//find vertex in clout from their ID
func findVertexBasedOnID(vertexid string, vertexes clout.Vertexes) *clout.Vertex {
	for ID, vertex := range vertexes {
		if vertexid == ID {
			return vertex
		}
	}
	return nil
}

// Return 'true' if 'propertyFilterConstraints' in substitution_filter of substitution_mappings matches with
// property value in abstract node template.
func checkForSubstitutionFilter(abstractVertex *clout.Vertex, substituteVertex *clout.Vertex, clout *clout.Clout) bool {
	var propertyFilterName string
	var propertyFilterValues []interface{}
	var propertyFilterConstraintName interface{}
	var propertyValueInAbstractNode interface{}

	substituteVertexProperties := substituteVertex.Properties
	substitutionFilters, _ := substituteVertexProperties["substitutionFilter"].([]interface{})

	if len(substitutionFilters) == 0 {
		return true
	}

	//get substitution_filter from substitute node template
	for _, substitutionFilter := range substitutionFilters {
		substitutionFilterMap, _ := substitutionFilter.(map[string]interface{})
		propertyFilterConstraints, _ := substitutionFilterMap["propertyFilterConstraints"].(map[string]interface{})

		for propertyName, propertyFilterConstraint := range propertyFilterConstraints {
			constraintsList, _ := propertyFilterConstraint.([]interface{})
			propertyFilterName = propertyName

			for _, propertyFilterConstraint := range constraintsList {
				propertyFilterConstraintMap, _ := propertyFilterConstraint.(map[string]interface{})
				functionCall, _ := propertyFilterConstraintMap["functionCall"].(map[string]interface{})
				propertyFilterConstraintName, _ = functionCall["name"]
				arguments, _ := functionCall["arguments"].([]interface{})

				for _, argument := range arguments {
					argumentMap, _ := argument.(map[string]interface{})
					propertyFilterValues = append(propertyFilterValues, argumentMap["value"])
				}
			}
		}
	}

	abstractVertexProperties := abstractVertex.Properties
	properties, _ := abstractVertexProperties["properties"].(map[string]interface{})

	// get property value of abstract node template
	for abstractVertexPropertyName, property := range properties {
		if abstractVertexPropertyName == propertyFilterName {
			var propertyValue interface{}
			propertyFilterConstraintMap, _ := property.(map[string]interface{})
			propertyValueInAbstractNode, _ = propertyFilterConstraintMap["value"]

			if propertyValueInAbstractNode != nil {
				break
			}

			functionCall, _ := propertyFilterConstraintMap["functionCall"].(map[string]interface{})
			propertyConstraintName, _ := functionCall["name"]
			arguments, _ := functionCall["arguments"].([]interface{})

			for _, argument := range arguments {
				argumentMap, _ := argument.(map[string]interface{})
				propertyValue, _ = argumentMap["value"]
			}

			if propertyConstraintName == "get_input" {
				cloutProperties := clout.Properties
				tosca := cloutProperties["tosca"].(map[string]interface{})
				inputs := tosca["inputs"].(map[string]interface{})

				for propName, property := range inputs {
					if propName == propertyValue {
						propertyMap, _ := property.(map[string]interface{})
						propertyValueInAbstractNode = propertyMap["value"]
					}
				}
			}
		}
	}

	// for now, handle "equal" constraint only. code will need to be added below for other
	// constraint value types such as "greater_than"
	if propertyFilterConstraintName == "equal" || propertyFilterConstraintName == "valid_values" {
		for _, propertyFilterValue := range propertyFilterValues {
			if propertyValueInAbstractNode == propertyFilterValue {
				return true
			}
		}
	}
	return false
}
