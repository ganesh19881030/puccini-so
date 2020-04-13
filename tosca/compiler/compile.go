package compiler

import (
	"strings"

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
			tr := clout_.NewVertex(clout.NewKey())
			SetMetadata(tr, "policyTrigger")

			if trigger.Operation != nil {
				operation := make(map[string]interface{})
				operation["description"] = trigger.Operation.Description
				operation["implementation"] = trigger.Operation.Implementation
				operation["dependencies"] = trigger.Operation.Dependencies
				operation["inputs"] = trigger.Operation.Inputs
				tr.Properties["operation"] = operation
			} else if trigger.Workflow != nil {
				wv := workflows[trigger.Workflow.Name]

				e := tr.NewEdgeTo(wv)
				SetMetadata(e, "policyTriggerWorkflow")
			}

			tr.Properties["name"] = trigger.Name
			tr.Properties["description"] = trigger.Description
			tr.Properties["event_type"] = trigger.EventType

			if trigger.Condition != nil {
				conditionClauses := make(map[string]normal.FunctionCalls)
				for name, conditionClause := range trigger.Condition.ConditionClauseConstraints {
					conditionClauses[name] = conditionClause
				}
				tr.Properties["condition"] = conditionClauses
			}

			if trigger.Action != nil {
				tr.Properties["action"] = trigger.Action
			}

			e := v.NewEdgeTo(tr)
			SetMetadata(e, "policyTrigger")
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

			for capabilityName, capability := range substitution.CapabilityMappings {
				nodeTemplate := capability.NodeTemplate
				vv := nodeTemplates[nodeTemplate.Name]
				e := v.NewEdgeTo(vv)

				SetMetadata(e, "capabilityMapping")
				e.Properties["capability"] = capability.Name
				e.Properties["capabilityName"] = capabilityName
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

	//store separate implementation vertexes for each abstarct vertex
	storeSubstituteVertexesForEachAbstractVertex(clout_, s)

	//attach properties of abstract vertexes to the substitute vertexes
	addPropertiesOfAbstractVertexInSubstituteVertexes(clout_)

	addNodeTemplateNameForDanglingRequirements(clout_)

	addNodeTemplateNameForFunctionCall(clout_)

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

//This method performs following operations for each abstract vertex:
// . Find its substitute vertexes
// . For each of these substitute vertexes, find its implementation nodeTemplates
//   and create new vertexes for each of them and store in clout
func storeSubstituteVertexesForEachAbstractVertex(cloutFile *clout.Clout, s *normal.ServiceTemplate) {
	cloutVertexes := cloutFile.Vertexes
	vertexesToDeleteFromClout := make(clout.Vertexes)

	for _, vertex := range cloutVertexes {
		var directive interface{} = "substitute"
		var directiveLists []string
		var substituteVertexIDs string
		var serviceTemp normal.ServiceTemplate
		var substitution *normal.Substitution
		nodeTemp := make(normal.NodeTemplates)

		// ignore vertexes other than "node-template"
		if !isVertexOfSpecificKind(vertex, "nodeTemplate") {
			continue
		}

		vertexProperties := vertex.Properties
		abstractVertexName, _ := vertexProperties["name"].(string)

		// abstract node found, look for its substitute directive and find vertexes of
		// implementation of abstract node template
		substituteVertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, vertexProperties)

		if len(substituteVertexes) == 0 {
			continue
		}

		// find implementation nodeTemplates
		serviceTemplate := s
		for _, substituteVertex := range substituteVertexes {

			properties := substituteVertex.Properties
			substituteVertexName, _ := properties["name"]

			if isVertexOfSpecificKind(substituteVertex, "nodeTemplate") {
				for nodeTemplateName, nodeTemplate := range serviceTemplate.NodeTemplates {
					if substituteVertexName == nodeTemplateName {
						nodeTemplateName = abstractVertexName + "." + substituteVertexName.(string)
						nodeTemplate.Name = nodeTemplateName
						nodeTemp[nodeTemplateName] = nodeTemplate
					}
				}
			} else if isVertexOfSpecificKind(substituteVertex, "substitution") {
				for _, substitute := range serviceTemplate.Substitution {
					if substitute.Type == properties["type"] {
						substitution = substitute
					}
				}
			}
		}

		// create new vertexes for substitute node templates and store in clout
		serviceTemp.Substitution = append(serviceTemp.Substitution, substitution)
		serviceTemp.NodeTemplates = nodeTemp
		cl, _ := Compile(&serviceTemp)

		//store newly created vertexes in old clout
		for ID, vertex := range cl.Vertexes {
			substituteVertexIDs = substituteVertexIDs + ":" + ID
			cloutFile.Vertexes[ID] = vertex
		}

		//append vertex IDs of substitute vertexes in substitute directives of abstract vertex
		directive = directive.(string) + substituteVertexIDs
		directiveLists = append(directiveLists, directive.(string))
		vertexProperties["directives"] = directiveLists

		for ID, vertex := range substituteVertexes {
			vertexesToDeleteFromClout[ID] = vertex
		}
	}

	//delete old substitute vertexes from main clout as they are no longer relevant
	for vertexID := range vertexesToDeleteFromClout {
		if _, ok := cloutFile.Vertexes[vertexID]; ok {
			delete(cloutFile.Vertexes, vertexID)
		}
	}
}

//This method copies property values from abstract node template to its implementation vertexes
func addPropertiesOfAbstractVertexInSubstituteVertexes(cloutFile *clout.Clout) {
	cloutVertexes := cloutFile.Vertexes

	for _, vertex := range cloutVertexes {
		// ignore vertexes other than "nodeTemplate"
		if !isVertexOfSpecificKind(vertex, "nodeTemplate") {
			continue
		}

		vertexProperties := vertex.Properties

		substituteVertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, vertexProperties)

		// if substitute vertexes are empty, its not an abstract node
		if len(substituteVertexes) == 0 {
			continue
		}

		//get properties of abstract node template
		propertiesOfAbstractNodeTemplate := make(map[string]interface{})
		properties, _ := vertexProperties["properties"].(map[string]interface{})

		for propertyName, property := range properties {
			propertyMap, _ := property.(map[string]interface{})
			propertyValue, _ := propertyMap["value"]
			if propertyValue != nil {
				propertiesOfAbstractNodeTemplate[propertyName] = property
			}
		}

		//if properties of abstract node template not available then continue
		if propertiesOfAbstractNodeTemplate == nil {
			continue
		}

		//copy properties of abstract node template to substitute vertexes
		for _, vertex := range substituteVertexes {
			vertexProperties := vertex.Properties
			vertexProps, _ := vertexProperties["properties"].(map[string]interface{})

			for _, property := range vertexProps {
				var propertyValue interface{}
				propertyMap, _ := property.(map[string]interface{})
				value, _ := propertyMap["value"]

				//in substitute vertex, if value is already given then continue
				if value != nil {
					continue
				}

				functionCall, _ := propertyMap["functionCall"].(map[string]interface{})
				propertyConstraintName, _ := functionCall["name"]
				arguments, _ := functionCall["arguments"].([]interface{})

				for _, argument := range arguments {
					argumentMap, _ := argument.(map[string]interface{})
					propertyValue, _ = argumentMap["value"]
				}

				//match name of property in abstract and substitute vertexes and assign property value
				propValue, _ := propertyValue.(string)
				if val, ok := propertiesOfAbstractNodeTemplate[propValue]; ok && propertyConstraintName.(string) != "" {
					delete(propertyMap, "functionCall")
					valMap, _ := val.(map[string]interface{})
					propertyValue, _ = valMap["value"]
					propertyMap["value"] = propertyValue
				}
			}
		}
	}
}

//find substitute vertexes from abstract vertex
func findSubstituteVertexesFromAbstractVertex(cloutVertexes clout.Vertexes, vertexProperties map[string]interface{}) clout.Vertexes {
	directiveList, _ := vertexProperties["directives"].([]string)

	if len(directiveList) == 0 {
		return nil
	}

	substituteVertexes := make(clout.Vertexes)
	for _, directive := range directiveList {

		if directive == "substitute" {
			continue
		}

		substituteDirective := strings.Split(directive, ":")
		for ind2, vertexID := range substituteDirective {
			if ind2 == 0 {
				continue
			}
			vertexFromClout := findVertexBasedOnID(vertexID, cloutVertexes)
			if vertexFromClout != nil {
				substituteVertexes[vertexFromClout.ID] = vertexFromClout
			}
		}
	}
	return substituteVertexes
}

//this method stores the nodeTemplateName for dangling requirements
func addNodeTemplateNameForDanglingRequirements(cloutFile *clout.Clout) {
	cloutVertexes := cloutFile.Vertexes

	//look for 'nodeTemplate' vertexes
	for vertexID, vertex := range cloutVertexes {
		if !isVertexOfSpecificKind(vertex, "nodeTemplate") {
			continue
		}

		vertexProperties := vertex.Properties
		directives, _ := vertexProperties["directives"].([]interface{})

		//ignore abstract vertexes
		if len(directives) != 0 {
			continue
		}

		//find vertexes whose nodeTemplateName is empty in requirements section
		requirements, _ := vertexProperties["requirements"].([]interface{})
		for _, requirement := range requirements {
			var requirementNames []string

			requirementMap := requirement.(map[string]interface{})
			nodeTemplateName := requirementMap["nodeTemplateName"]
			nodeTypeName := requirementMap["nodeTypeName"]

			//if nodeTemplateName for requirement is not empty then it is not a dangling requirement
			if nodeTemplateName != "" {
				continue
			}

			//find substitution mapping vertex from the current vertex
			substitutionMappingVertex := findSubstitutionMappingVertexBasedOnVertexID(cloutVertexes, vertexID)
			if substitutionMappingVertex == nil {
				continue
			}

			//store requirementNames from substitution mapping vertex after matching with vertexID
			for _, edge := range substitutionMappingVertex.EdgesOut {
				metadata := edge.Metadata
				kindData := metadata["puccini-tosca"].(map[string]interface{})
				kind := kindData["kind"]
				if edge.TargetID == vertexID && kind == "requirementMapping" {
					edgeProps := edge.Properties
					requirementNames = append(requirementNames, edgeProps["requirementName"].(string))
				}
			}

			//find abstract vertex from substitute vertex
			for _, requirementName := range requirementNames {
				var capabilityName string
				var capabilityTypeName string
				var targetVertex *clout.Vertex

				abstractVertex := findAbstractVertexFromSubstituteVertex(cloutVertexes, vertexID)
				if abstractVertex == nil {
					continue
				}

				abstractVertexProperties := abstractVertex.Properties
				abstractVertexRequirements := abstractVertexProperties["requirements"].([]interface{})

				// find target vertex, capabilityName and its capabilityTypeName after matching abstract vertex's
				//    requirement name with requirements of substitutionMapping vertex
				for _, requirement := range abstractVertexRequirements {
					requirementData := requirement.(map[string]interface{})
					abstractVertexRequirementName := requirementData["name"]

					if abstractVertexRequirementName == requirementName {
						targetVertex = findVertexBasedOnName(requirementData["nodeTemplateName"].(string), cloutVertexes)
						capabilityName = requirementData["capabilityName"].(string)
						capabilityTypeName = requirementData["capabilityTypeName"].(string)
					}
				}

				if targetVertex == nil {
					continue
				}

				targetSubstituteVertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, targetVertex.Properties)

				//if capability name is empty then add nodeTemplateName based on CapabilityType or based on Requirements
				if capabilityName == "" {

					var isNodeTypeMatch bool = false
					targetVertexProps := targetVertex.Properties
					targetVertexTypes, _ := targetVertexProps["types"].(map[string]interface{})

					//if nodeType in target vertex match with nodeType of requirement then add nodeTemplateName
					// based on CapabilityType of requirement
					for nodeType := range targetVertexTypes {
						if nodeType == nodeTypeName {
							isNodeTypeMatch = true
							addNodeTemplateNameBasedOnCapabilityType(targetSubstituteVertexes, capabilityTypeName, requirementMap, targetVertex)
						}
					}
					if !isNodeTypeMatch {
						//if nodeType in target vertex did not match with nodeType of requirement then add nodeTemplateName
						// based on requirement's nodeType
						addNodeTemplateNameBasedOnNodeType(targetSubstituteVertexes, nodeTypeName, requirementMap)
					}
				} else {
					//if capabilityName is given for requirement in abstract vertex then add nodeTemplateName based on capabilityName
					addNodeTemplateNameBasedOnCapabilityName(targetSubstituteVertexes, capabilityName, requirementMap, requirementName)
				}
			}
		}
	}
}

//match nodeType of requirement and nodeType of vertex in target substitute vertexes then add nodeTemplate for requirement
func addNodeTemplateNameBasedOnNodeType(targetSubstituteVertexes clout.Vertexes, nodeTypeName interface{},
	requirementMap map[string]interface{}) {
	for _, vertex := range targetSubstituteVertexes {

		if !isVertexOfSpecificKind(vertex, "nodeTemplate") {
			continue
		}

		vertexProps := vertex.Properties
		types, _ := vertexProps["types"].(map[string]interface{})
		for nodeType := range types {
			if nodeType == nodeTypeName {
				requirementMap["nodeTemplateName"] = vertexProps["name"]
			}
		}
	}
}

//match capabilityName in requirement of abstract vertex with capabilityName in substitution mapping vertex
// of target vertex then add nodeTemplateName
func addNodeTemplateNameBasedOnCapabilityName(targetSubstituteVertexes clout.Vertexes, capabilityName string,
	requirementMap map[string]interface{}, requirementName string) {
	for _, vertex := range targetSubstituteVertexes {
		if !isVertexOfSpecificKind(vertex, "substitution") {
			continue
		}

		for _, edge := range vertex.EdgesOut {
			property := edge.Properties
			metadata := edge.Metadata
			kindData := metadata["puccini-tosca"].(map[string]interface{})
			kind := kindData["kind"]

			if kind == "capabilityMapping" && capabilityName == property["capabilityName"] {
				for VertexID, targetVertex := range targetSubstituteVertexes {
					if VertexID == edge.TargetID && ((requirementMap["name"] == requirementName) || requirementName == "") {
						props := targetVertex.Properties
						requirementMap["nodeTemplateName"] = props["name"]
					}
				}
			}
		}
	}
}

//match capabilityType of requirement with capabilityType of target substitute vertex then add nodeTemplateName
func addNodeTemplateNameBasedOnCapabilityType(targetSubstituteVertexes clout.Vertexes, capabilityTypeName string,
	requirementMap map[string]interface{}, targetVertex *clout.Vertex) {
	var targetCapabilityName string
	targetVertexProps := targetVertex.Properties
	targetVertexCapabilities, _ := targetVertexProps["capabilities"].(map[string]interface{})

	for capabilityName, capability := range targetVertexCapabilities {
		capabilityData := capability.(map[string]interface{})
		types, _ := capabilityData["types"].(map[string]interface{})

		for capabilityType := range types {
			if capabilityType == capabilityTypeName {
				targetCapabilityName = capabilityName
			}
		}
	}

	if targetCapabilityName != "" {
		addNodeTemplateNameBasedOnCapabilityName(targetSubstituteVertexes, targetCapabilityName, requirementMap, "")
	}
}

func findSubstitutionMappingVertexBasedOnVertexID(cloutVertexes clout.Vertexes, vertexID string) *clout.Vertex {
	abstractVertex := findAbstractVertexFromSubstituteVertex(cloutVertexes, vertexID)
	if abstractVertex == nil {
		return nil
	}

	vertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, abstractVertex.Properties)

	for _, substitutionMappingVertex := range vertexes {
		if isVertexOfSpecificKind(substitutionMappingVertex, "substitution") {
			return substitutionMappingVertex
		}
	}
	return nil
}

func findAbstractVertexFromSubstituteVertex(cloutVertexes clout.Vertexes, substituteVertexID string) *clout.Vertex {
	for _, abstractVertex := range cloutVertexes {
		if !isVertexOfSpecificKind(abstractVertex, "nodeTemplate") {
			continue
		}

		vertexProperties := abstractVertex.Properties
		substituteVertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, vertexProperties)

		for VertexID := range substituteVertexes {
			if VertexID == substituteVertexID {
				return abstractVertex
			}
		}
	}
	return nil
}

//find vertex in clout from their Name
func findVertexBasedOnName(vertexName string, vertexes clout.Vertexes) *clout.Vertex {
	for _, vertex := range vertexes {
		prop := vertex.Properties
		nodeTemplateName, _ := prop["name"]
		if nodeTemplateName == vertexName {
			return vertex
		}
	}
	return nil
}

// Add nodeTemplateName for 'get_attribute' functionCall in clout
func addNodeTemplateNameForFunctionCall(cloutFile *clout.Clout) {
	cloutVertexes := cloutFile.Vertexes

	for _, vertex := range cloutVertexes {
		if !isVertexOfSpecificKind(vertex, "nodeTemplate") {
			continue
		}

		vertexProperties := vertex.Properties
		abstractVertexName, _ := vertexProperties["name"].(string)

		// abstract node found, look for its substitute directive and find vertexes of
		// implementation of abstract node template
		substituteVertexes := findSubstituteVertexesFromAbstractVertex(cloutVertexes, vertexProperties)

		for _, substituteVertex := range substituteVertexes {
			substituteVertexProperties := substituteVertex.Properties
			substituteVertexName, _ := substituteVertexProperties["name"].(string)

			cloutProps, _ := cloutFile.Properties["tosca"].(map[string]interface{})
			cloutOutputs, _ := cloutProps["outputs"].(map[string]interface{})

			// add abstract nodeTemplateName with nodeTemplateName for get_attribute function
			// for outputs in topology_template
			for _, output := range cloutOutputs {
				outputData, _ := output.(map[string]interface{})
				functionCall, _ := outputData["functionCall"].(map[string]interface{})
				FunctionCallName, _ := functionCall["name"]

				arguments, _ := functionCall["arguments"].([]interface{})
				argument, _ := arguments[0].(map[string]interface{})
				nodeTemplateName, _ := argument["value"].(string)

				if strings.Contains(substituteVertexName, nodeTemplateName) && nodeTemplateName != "SELF" &&
					FunctionCallName == "get_attribute" {
					argument["value"] = abstractVertexName + "." + nodeTemplateName
				}
			}

			// add abstract nodeTemplateName for get_attribute function in attributes
			// of capabilities in node_template
			// Need to look at this logic again. It is highly dependent on the order in which
			// attribute value function arguments are stored
			capabilities, _ := substituteVertexProperties["capabilities"].(map[string]interface{})
			for ckey, capability := range capabilities {
				_ = ckey
				capabilityData := capability.(map[string]interface{})
				attributes := capabilityData["attributes"].(map[string]interface{})

				for _, attribute := range attributes {
					attributeData, _ := attribute.(map[string]interface{})
					functionCall, _ := attributeData["functionCall"].(map[string]interface{})
					FunctionCallName, _ := functionCall["name"]

					arguments, _ := functionCall["arguments"].([]interface{})
					var nodeTemplateName string
					var argument ard.Map

					// look for SELF in the function arguments in any order
					for _, arg := range arguments {
						if name, ok := arg.(ard.Map)["value"]; ok {
							argument = arg.(ard.Map)
							nodeTemplateName, ok = name.(string)
							if ok && nodeTemplateName == "SELF" {
								break
							}
						}

					}
					if nodeTemplateName != "SELF" {
						// look at the first argument
						argument, _ = arguments[0].(map[string]interface{})
						nodeTemplateName, _ = argument["value"].(string)

						if FunctionCallName == "get_attribute" {
							argument["value"] = abstractVertexName + "." + nodeTemplateName
						}
					}
				}
			}
		}
	}
}
