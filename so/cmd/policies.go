package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
)

var policies string

func init() {
	rootCmd.AddCommand(policiesCmd)
	policiesCmd.Flags().StringVarP(&policies, "policies-output", "w", "", "output policies steps data to file or directory (default is stdout)")
}

var policiesCmd = &cobra.Command{
	Use:   "policies [clout file PATH or URL] ",
	Short: "Create Policies steps from Clout",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		var path string
		if len(args) == 1 {
			path = args[0]
		}

		cloutFile, err := ReadClout(path)
		common.FailOnError(err)

		//create policy steps
		policy := createPolicies(cloutFile)

		// save policy steps in file
		if policies != "" {
			err = format.WriteOrPrint(policy, ardFormat, pretty, policies)
			common.FailOnError(err)
		}
	},
}

//create policy steps
func createPolicies(cloutFile *clout.Clout) *Policies {
	policy := NewPolicies()

	for _, vertex := range cloutFile.Vertexes {
		policyDef := NewPolicyDefinitions()

		//ignore vertexes other than policy
		if !isVertexOfSpecificKind(vertex.Metadata, "policy") {
			continue
		}

		policyName, _ := vertex.Properties["name"].(string)

		assignPropertyValuesInPoliciesFromInputs(vertex, cloutFile)

		createPolicyStep("start", vertex, cloutFile.Vertexes, policyDef)

		createPolicyStep("stop", vertex, cloutFile.Vertexes, policyDef)

		//create steps for triggers in policies
		createPolicyTriggersSteps(vertex, cloutFile.Vertexes, policyDef)

		policyDef.Name = policyName
		policyDef.Description = "policy " + policyName
		policy.PolicyDefinitions[policyName] = policyDef
	}
	return policy
}

func createPolicyStep(stepName string, vertex *clout.Vertex, Vertexes clout.Vertexes, policyDef *PolicyDefinition) {
	policyStep := NewPolicyStepDefinition()
	activities := NewPolicyActivityDefinition()

	policyName, _ := vertex.Properties["name"].(string)
	targetNodeTemplateOrGroupNames := findTargetVertexesForPolicy(vertex, Vertexes)

	//store policy step name
	policyStep.Name = policyName + "." + stepName

	//store policy 'target' node templates and groups
	policyStep.TargetNodeTemplateOrGroupNames = targetNodeTemplateOrGroupNames

	//store policy step operation
	activities.CallOperation = stepName

	//store policy step actions
	policyStep.Actions = append(policyStep.Actions, activities)

	//if step name is start then add onSuccess steps names for start policy
	if stepName == "start" {
		for _, edge := range vertex.EdgesOut {

			if !isVertexOfSpecificKind(edge.Metadata, "policyTrigger") {
				continue
			}

			triggerVertex := findVertexBasedOnID(edge.TargetID, Vertexes)
			triggerName, _ := triggerVertex.Properties["name"].(string)
			onSuccessStepName := policyName + "." + "trigger" + "." + triggerName
			policyStep.OnSuccessSteps = append(policyStep.OnSuccessSteps, onSuccessStepName)
		}
	}

	policyDef.Steps[policyStep.Name] = policyStep
}

//this method creates steps for policy triggers
func createPolicyTriggersSteps(vertex *clout.Vertex, Vertexes clout.Vertexes, policyDef *PolicyDefinition) {
	properties := vertex.Properties
	policyName, _ := properties["name"].(string)

	//find target vertex for policy
	targetNodeTemplateOrGroupNames := findTargetVertexesForPolicy(vertex, Vertexes)

	for _, edge := range vertex.EdgesOut {
		policyStep := NewPolicyStepDefinition()

		if !isVertexOfSpecificKind(edge.Metadata, "policyTrigger") {
			continue
		}
		//find trigger vertex
		triggerVertex := findVertexBasedOnID(edge.TargetID, Vertexes)
		triggerVertexProperties := triggerVertex.Properties

		//store trigger step name
		triggerVertexName, _ := triggerVertexProperties["name"].(string)
		stepName := policyName + "." + "trigger" + "." + triggerVertexName
		policyStep.Name = stepName

		//store trigger event type
		policyStep.EventType, _ = triggerVertexProperties["event_type"].(string)

		//store target node template or group names
		policyStep.TargetNodeTemplateOrGroupNames = targetNodeTemplateOrGroupNames

		//store trigger conditions
		conditions, _ := triggerVertexProperties["condition"].(map[string]interface{})
		assignPropertyValuesInTriggersFromPolicy(conditions, vertex)

		for conditionName, condition := range conditions {
			conditionMap := make(map[string]interface{})
			conditionNestedMap := make(map[string]map[string]interface{})
			conditionList, _ := condition.([]interface{})

			for _, conditionData := range conditionList {
				conditionDataMap, _ := conditionData.(map[string]interface{})
				functionCall, _ := conditionDataMap["functionCall"].(map[string]interface{})
				constraintClauseName, _ := functionCall["name"].(string)
				value, _ := conditionDataMap["value"]
				conditionMap[constraintClauseName] = value
				conditionNestedMap[conditionName] = conditionMap
			}
			policyStep.Conditions = append(policyStep.Conditions, &conditionNestedMap)
		}

		//store trigger actions
		actions, _ := triggerVertexProperties["action"].(map[string]interface{})
		updateAction, _ := actions["update"].(map[string]interface{})
		assignPropertyValuesInTriggersFromPolicy(updateAction, vertex)

		activities := NewPolicyActivityDefinition()
		for actionName, action := range updateAction {
			actionMap := make(map[string]interface{})
			actionData, _ := action.(map[string]interface{})
			value, _ := actionData["value"]
			actionMap[actionName] = value
			activities.Update = actionMap
		}
		policyStep.Actions = append(policyStep.Actions, activities)

		policyDef.Steps[policyStep.Name] = policyStep
	}
}

//this method assigns values to "condition" and "action" from properties of policy
func assignPropertyValuesInTriggersFromPolicy(dataMap map[string]interface{}, policyVertex *clout.Vertex) {
	var argumentValues []interface{}
	var propertyName string

	for _, data := range dataMap {
		functionList, _ := data.([]interface{})

		if functionList == nil {
			functionList = append(functionList, data.(map[string]interface{}))
		}

		for _, function := range functionList {
			functionMap, _ := function.(map[string]interface{})

			//if property value is found, then don't need to look further
			value, _ := functionMap["value"]
			if value != nil {
				continue
			}

			functionCall, _ := functionMap["functionCall"].(map[string]interface{})
			arguments, _ := functionCall["arguments"].([]interface{})

			for _, argument := range arguments {
				argumentMap, _ := argument.(map[string]interface{})
				argumentValues = append(argumentValues, argumentMap["value"])
			}

			for _, argumentValue := range argumentValues {
				if argumentValue.(string) == "SELF" {
					continue
				}
				propertyName, _ = argumentValue.(string)
			}

			//get the property value from policy properties
			if propertyName != "" {
				props, _ := policyVertex.Properties["properties"].(map[string]interface{})

				if prop, ok := props[propertyName]; ok {
					propData, _ := prop.(map[string]interface{})
					value, _ := propData["value"]
					if value != nil {
						functionMap["value"] = value
					}
				}
			}
		}
	}
}

//this method assigns property values to the policy's properties from clout properties
func assignPropertyValuesInPoliciesFromInputs(policyVertex *clout.Vertex, cloutFile *clout.Clout) {
	policyVertexProperties, _ := policyVertex.Properties["properties"].(map[string]interface{})

	for _, property := range policyVertexProperties {
		var argumentValues []interface{}
		propertyMap, _ := property.(map[string]interface{})
		value, _ := propertyMap["value"]

		//if property value found in policies then continue
		if value != nil {
			continue
		}

		functionCall, _ := propertyMap["functionCall"].(map[string]interface{})
		functionName, _ := functionCall["name"]
		arguments, _ := functionCall["arguments"].([]interface{})

		for _, argument := range arguments {
			argumentMap, _ := argument.(map[string]interface{})
			argumentValues = append(argumentValues, argumentMap["value"])
		}

		//assign property value to policy from inputs in clout
		if functionName == "get_input" && len(argumentValues) < 2 {
			for _, argumentValue := range argumentValues {

				cloutProperties, _ := cloutFile.Properties["tosca"].(map[string]interface{})
				inputs, _ := cloutProperties["inputs"].(map[string]interface{})

				if cloutProperty, ok := inputs[argumentValue.(string)]; ok {
					cloutPropertyMap, _ := cloutProperty.(map[string]interface{})
					propertyMap["value"] = cloutPropertyMap["value"]
					delete(propertyMap, "functionCall")
				}
			}
		}
	}
}

//find "target" vertexes for policy
func findTargetVertexesForPolicy(vertex *clout.Vertex, vertexes clout.Vertexes) []string {
	var targetVertexNames []string
	for _, edge := range vertex.EdgesOut {
		if isVertexOfSpecificKind(edge.Metadata, "nodeTemplateTarget") || isVertexOfSpecificKind(edge.Metadata, "groupTarget") {
			targetVertex := findVertexBasedOnID(edge.TargetID, vertexes)
			targetVertexName, _ := targetVertex.Properties["name"].(string)
			targetVertexNames = append(targetVertexNames, targetVertexName)
		}
	}
	return targetVertexNames
}

func isVertexOfSpecificKind(vertexMetadata map[string]interface{}, vertexKind string) bool {
	if vertexMetadata != nil {
		metaDataName, _ := vertexMetadata["puccini-tosca"].(map[string]interface{})
		kindName, _ := metaDataName["kind"]
		if kindName == vertexKind {
			return true
		}
	}
	return false
}

func findVertexBasedOnID(vertexid string, vertexes clout.Vertexes) *clout.Vertex {
	for ID, vertex := range vertexes {
		if vertexid == ID {
			return vertex
		}
	}
	return nil
}

// Policies ...
type Policies struct {
	PolicyDefinitions PolicyDefinitions `json:"policies" yaml:"policies"`
}

// NewPolicies ...
func NewPolicies() *Policies {
	return &Policies{
		PolicyDefinitions: make(PolicyDefinitions, 0),
	}
}

// PolicyDefinition ...
type PolicyDefinition struct {
	Name        string                `json:"name" yaml:"name"`
	Description string                `json:"description" yaml:"description"`
	Steps       PolicyStepDefinitions `json:"steps" yaml:"steps"`
}

// NewPolicyDefinitions ...
func NewPolicyDefinitions() *PolicyDefinition {
	return &PolicyDefinition{
		Steps: make(PolicyStepDefinitions),
	}
}

// PolicyDefinitions ...
type PolicyDefinitions map[string]*PolicyDefinition

// PolicyStepDefinition ...
type PolicyStepDefinition struct {
	Name string `json:"name" yaml:"name"`

	EventType                      string                               `json:"event" yaml:"event"`
	Conditions                     []*map[string]map[string]interface{} `json:"condition" yaml:"condition"`
	TargetNodeTemplateOrGroupNames []string                             `json:"target" yaml:"target"`
	Actions                        PolicyActivityDefinitions            `json:"action" yaml:"action"`
	OnSuccessSteps                 []string                             `json:"on_success" yaml:"on_success"`
	OnFailureSteps                 []string                             `json:"on_failure" yaml:"on_failure"`
}

// NewPolicyStepDefinition ...
func NewPolicyStepDefinition() *PolicyStepDefinition {
	return &PolicyStepDefinition{
		Actions: make(PolicyActivityDefinitions, 0),
	}
}

// PolicyStepDefinitions ...
type PolicyStepDefinitions map[string]*PolicyStepDefinition

// PolicyActivityDefinition ...
type PolicyActivityDefinition struct {
	Step             string                 `json:"-" yaml:"-"`
	DelegateWorkflow *WorkflowDefinition    `json:"-" yaml:"-"`
	InlineWorkflow   *WorkflowDefinition    `json:"-" yaml:"-"`
	CallOperation    string                 `json:"call_operation" yaml:"call_operation"`
	Update           map[string]interface{} `json:"update" yaml:"update"`
}

// NewPolicyActivityDefinition ...
func NewPolicyActivityDefinition() *PolicyActivityDefinition {
	return &PolicyActivityDefinition{}
}

// PolicyActivityDefinitions ...
type PolicyActivityDefinitions []*PolicyActivityDefinition
