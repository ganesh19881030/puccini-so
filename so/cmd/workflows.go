package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
)

const version = "1.0"

var targetNodeOperations map[string]string
var sourceNodeOperations map[string]string

var workflows string

func init() {
	rootCmd.AddCommand(workflowsCmd)
	workflowsCmd.Flags().StringVarP(&workflows, "workflows-output", "w", "", "output workflow steps data to file or directory (default is stdout)")
	workflowsCmd.Flags().StringVarP(&output, "output", "o", "", "output clout data to file or directory (default is stdout)")
}

var workflowsCmd = &cobra.Command{
	Use:   "workflows [clout file PATH or URL] ",
	Short: "Create Workflows steps from Clout",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		var path string
		if len(args) == 1 {
			path = args[0]
		}

		clout, err := ReadClout(path)
		common.FailOnError(err)

		// create workflow steps
		workFlows := createWorkFlows(clout)

		// store workflow steps into clout structure
		storeWorkflowsIntoClout(workFlows.WorkflowDefinitions, clout)

		// save/print clout file
		if !common.Quiet || (output != "") {
			err = format.WriteOrPrint(clout, ardFormat, pretty, output)
			common.FailOnError(err)
		}

		// save/print workflow steps file
		if workflows != "" {
			err = format.WriteOrPrint(workFlows, ardFormat, pretty, workflows)
			common.FailOnError(err)
		}
	},
}

// create workflow steps by analyzing requirements of node template vertexes present
// in the input clout file
func createWorkFlows(c *clout.Clout) *Workflows {

	workFlows := NewWorkflows()
	workFlowDef := NewWorkflowDefinition()

	cloutVertexes := c.Vertexes

	// [TOSCA-Simple-Profile-YAML-v1.3] @  5.8.5
	// create maps for target and source node operations
	targetNodeOperations = make(map[string]string)
	targetNodeOperations["create"] = "pre_configure_target"
	targetNodeOperations["configure"] = "post_configure_target"

	sourceNodeOperations = make(map[string]string)
	sourceNodeOperations["create"] = "pre_configure_source"
	sourceNodeOperations["configure"] = "post_configure_source"
	sourceNodeOperations["start"] = "add_source"

	// loop through all vertexes in the input clout file
	for _, vertex := range cloutVertexes {

		// ignore vertexes other than "node-template"
		if !isVertexNodeTemplate(vertex) {
			continue
		}

		// get vertex properties
		vertexProperties := vertex.Properties

		if vertexProperties != nil {

			// getvertex name
			vertexName := vertexProperties["name"].(string)

			// get vertex requirements
			requirements := vertexProperties["requirements"].([]interface{})
			requirementLength := len(requirements)

			// get interface operations of vertex
			standardOperations := getInterfacesOperationsOfVertex(vertexProperties)

			for _, operationName := range standardOperations {
				// delete and stop operations are currently not used/supported
				if (operationName == "delete") || (operationName == "stop") {
					continue
				}

				// if a node has no requirements, its a target node. otherwise, its a source node.
				if requirementLength == 0 {
					configureName := targetNodeOperations[operationName]
					createTargetNodeWorkFlowSteps(vertexName, cloutVertexes, workFlowDef, operationName, configureName)
				} else {
					configureName := sourceNodeOperations[operationName]
					createSourceNodeWorkFlowSteps(vertexName, requirements, workFlowDef, "Standard", operationName, cloutVertexes, configureName, "")
				}
			}
		}
	}

	// save all the workflow steps under workflow named deploy
	workFlowDef.Name = "deploy"
	workFlowDef.Description = "workflow deploy"
	workFlows.WorkflowDefinitions[workFlowDef.Name] = workFlowDef

	return workFlows
}

// create workflow steps for a target node
func createTargetNodeWorkFlowSteps(vertexName string, cloutVertexes clout.Vertexes,
	workFlowsDef *WorkflowDefinition, standardOperationName string, configureName string) error {

	// create new workflow steps definition
	workFlowStepsDef := NewWorkflowStepDefinition()

	// create new workflow activity definition
	workFlowActivityDef := NewWorkflowActivityDefinition()

	// create workflow step name
	workFlowStepName := vertexName + "_" + standardOperationName

	// set 'target' property of work flow step
	workFlowStepsDef.TargetNodeTemplateOrGroupName = vertexName

	// find vertexes which have this node as a requirement
	for _, vertex := range cloutVertexes {

		// ignore vertexes other than "node-template"
		if !isVertexNodeTemplate(vertex) {
			continue
		}

		// get vertex properties
		vertexProperties := vertex.Properties

		if vertexProperties != nil {
			vertexRequirements := vertexProperties["requirements"].([]interface{})
			sourceVertextName := vertexProperties["name"].(string)
			length := len(vertexRequirements)
			if length != 0 {
				for _, vertexRequirement := range vertexRequirements {
					if vertexRequirement != nil {
						vertexRequirementMap := vertexRequirement.(map[string]interface{})
						vertexRequirementName := vertexRequirementMap["name"]

						// if this vertex has a requirement for the target vertex, create its workflow steps
						if vertexRequirementName == vertexName {

							onSuccessName := sourceVertextName + "_" + vertexName + "_" + configureName

							if standardOperationName == "start" {
								// for start, don't need any more steps
								onSuccessName = sourceVertextName + "_" + "create"
							} else {
								// for create and configure, create steps
								createOnSuccessStepsForTarget(sourceVertextName, vertexName, configureName, onSuccessName, workFlowsDef)
							}
							workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, onSuccessName)
						}
					}
				}
			}
		}
	}
	workFlowActivityDef.CallOperation = "Standard." + standardOperationName
	workFlowStepsDef.Activities = append(workFlowStepsDef.Activities, workFlowActivityDef)
	workFlowStepsDef.Name = workFlowStepName
	workFlowsDef.Steps[workFlowStepName] = workFlowStepsDef

	return nil
}

// create onSuccess steps for target node
func createOnSuccessStepsForTarget(targetNodeName string, targetNodeRequirementName string,
	activityName string, keyName string, workFlowsDef *WorkflowDefinition) *WorkflowStepDefinition {

	// create new workflow steps definition
	workFlowStepsDef := NewWorkflowStepDefinition()

	// create new workflow activity definition
	workFlowActivityDef := NewWorkflowActivityDefinition()
	workFlowActivityDef.CallOperation = "Configure." + activityName
	workFlowStepsDef.Activities = append(workFlowStepsDef.Activities, workFlowActivityDef)

	// set 'target' property of work flow step
	workFlowStepsDef.TargetNodeTemplateOrGroupName = targetNodeName
	workFlowStepsDef.TargetNodeRequirementName = targetNodeRequirementName

	var onSuccessName string
	if activityName == "pre_configure_target" {
		onSuccessName = targetNodeRequirementName + "_" + "configure"
	} else if activityName == "post_configure_target" {
		onSuccessName = targetNodeRequirementName + "_" + "start"
	}

	workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, onSuccessName)
	workFlowStepsDef.OperationHost = "TARGET"
	workFlowStepsDef.Name = keyName
	workFlowsDef.Steps[keyName] = workFlowStepsDef

	return workFlowStepsDef
}

// create workflow steps for a source node
func createSourceNodeWorkFlowSteps(vertexName string, requirements []interface{}, workFlowsDef *WorkflowDefinition,
	operationType string, operationName string, cloutVertexes clout.Vertexes, configureName string, key string) error {

	workFlowStepsDef := NewWorkflowStepDefinition()
	workFlowActivityDef := NewWorkflowActivityDefinition()

	var keyName string
	if key != "" {
		keyName = key
	} else {
		keyName = vertexName + "_" + operationName
	}

	workFlowActivityDef.CallOperation = operationType + "." + operationName
	workFlowStepsDef.Activities = append(workFlowStepsDef.Activities, workFlowActivityDef)

	createOnSuccessStepsForSource(vertexName, cloutVertexes, workFlowsDef, workFlowStepsDef, operationName, targetNodeOperations[operationName])

	var vertexRequirementName interface{}
	var endSuccessName string
	var onSuccessName string

	// loop through requirements and create steps for each requirement
	for _, requirement := range requirements {
		if requirement != nil {

			vertexRequirementMap := requirement.(map[string]interface{})
			vertexRequirementName = vertexRequirementMap["name"]

			onSuccessName = vertexName + "_" + vertexRequirementName.(string) + "_" + configureName
			workFlowStepsDef.TargetNodeTemplateOrGroupName = vertexName
			workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, onSuccessName)
			workFlowStepsDef.Name = keyName

			if operationName == "add_target" {
				workFlowStepsDef.OperationHost = "SOURCE"
			} else if operationName == "add_source" {
				workFlowStepsDef.OperationHost = "TARGET"
			}

			workFlowsDef.Steps[keyName] = workFlowStepsDef

			if operationName == "start" {
				// create add_source and add_target steps for start operation's onSucess
				targetSuccessName := vertexName + "_" + vertexRequirementName.(string) + "_" + "add_target"
				workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, targetSuccessName)
				workFlowsDef.Steps[keyName] = workFlowStepsDef

				createSourceNodeWorkFlowSteps(vertexName, requirements, workFlowsDef, "Configure", "add_source", cloutVertexes, "add_source", onSuccessName)
				createSourceNodeWorkFlowSteps(vertexName, requirements, workFlowsDef, "Configure", "add_target", cloutVertexes, "add_target", targetSuccessName)
				break
			}

			// create workflow steps for the requirement
			workFlowStepsDef = NewWorkflowStepDefinition()
			workFlowActivityDef = NewWorkflowActivityDefinition()

			keyName = onSuccessName
			workFlowActivityDef.CallOperation = "Configure." + configureName
			workFlowStepsDef.Activities = append(workFlowStepsDef.Activities, workFlowActivityDef)
			workFlowStepsDef.TargetNodeRequirementName = vertexRequirementName.(string)
			workFlowStepsDef.OperationHost = "SOURCE"

			if operationName == "add_target" {
				workFlowStepsDef.OperationHost = "SOURCE"
			} else if operationName == "add_source" {
				workFlowStepsDef.OperationHost = "TARGET"
			}
		}
	}

	if operationName == "create" {
		endSuccessName = vertexName + "_" + "configure"
	} else if operationName == "configure" {
		endSuccessName = vertexName + "_" + "start"
	}

	if endSuccessName != "" {
		workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, endSuccessName)
	}

	workFlowStepsDef.TargetNodeTemplateOrGroupName = vertexName
	workFlowStepsDef.Name = keyName
	workFlowsDef.Steps[keyName] = workFlowStepsDef

	return nil
}

// create onSuccess steps for source nodes
func createOnSuccessStepsForSource(vertexName interface{}, cloutVertexes clout.Vertexes, workFlowsDef *WorkflowDefinition,
	workFlowStepsDefCalled *WorkflowStepDefinition, standardOperationName string, configureName string) error {

	// create new workflow steps definition
	workFlowStepsDef := NewWorkflowStepDefinition()

	// create new workflow activity definition
	workFlowActivityDef := NewWorkflowActivityDefinition()

	// scan all vertexes and create onSuccess workflow steps
	for _, vertex := range cloutVertexes {

		// ignore vertexes other than "node-template"
		if !isVertexNodeTemplate(vertex) {
			continue
		}

		// get vertex properties
		vertexProperties := vertex.Properties

		// find vertexes which have requirements (i.e. source nodes)
		if vertexProperties != nil {
			vertexRequirements := vertexProperties["requirements"].([]interface{})
			sourceVertextName := vertexProperties["name"].(string)
			length := len(vertexRequirements)

			if length != 0 {
				for _, vertexRequirement := range vertexRequirements {
					if vertexRequirement != nil {

						vertexRequirementMap := vertexRequirement.(map[string]interface{})
						vertexRequirementName := vertexRequirementMap["name"]
						nodeTemplateName := vertexRequirementMap["nodeTemplateName"]

						workFlowStepsDef = NewWorkflowStepDefinition()
						workFlowActivityDef = NewWorkflowActivityDefinition()

						// if the source vertex name is a requirement for this vertex,
						// create its workflow steps
						if nodeTemplateName == vertexName {

							var onSuccessName string

							if standardOperationName == "create" {
								onSuccessName = vertexName.(string) + "_" + "configure"
							} else if standardOperationName == "configure" {
								onSuccessName = vertexName.(string) + "_" + "start"
							} else {
								if standardOperationName == "start" {
									onSuccessName = sourceVertextName + "_" + "create"
									workFlowStepsDefCalled.OnSuccessSteps = append(workFlowStepsDefCalled.OnSuccessSteps, onSuccessName)
								}
								continue
							}

							// create workflow steps for source node's onSuccess
							workFlowStepsDef.TargetNodeTemplateOrGroupName = sourceVertextName
							workFlowStepsDef.TargetNodeRequirementName = vertexRequirementName.(string)
							workFlowStepsDef.OperationHost = "TARGET"
							workFlowActivityDef.CallOperation = "Configure." + configureName
							workFlowStepsDef.Activities = append(workFlowStepsDef.Activities, workFlowActivityDef)
							workFlowStepsDef.OnSuccessSteps = append(workFlowStepsDef.OnSuccessSteps, onSuccessName)

							workFlowStepName := sourceVertextName + "_" + vertexRequirementName.(string) + "_" + configureName
							workFlowStepsDef.Name = workFlowStepName
							workFlowsDef.Steps[workFlowStepName] = workFlowStepsDef

							workFlowStepsDefCalled.OnSuccessSteps = append(workFlowStepsDefCalled.OnSuccessSteps, workFlowStepName)
						}
					}
				}
			}
		}
	}

	return nil
}

// add all the created workflow steps into the input clout structure
func storeWorkflowsIntoClout(Workflows WorkflowDefinitions, cloutP *clout.Clout) error {

	nodeTemplates := make(map[string]*clout.Vertex)
	cloutVertexes := cloutP.Vertexes

	for _, vertex := range cloutVertexes {
		// ignore vertexes other than "node-template"
		if !isVertexNodeTemplate(vertex) {
			continue
		}
		vertexProperties := vertex.Properties
		vertexName := vertexProperties["name"]
		nodeTemplates[vertexName.(string)] = vertex
	}

	workflows := make(map[string]*clout.Vertex)

	// Workflows
	for _, workflow := range Workflows {
		v := cloutP.NewVertex(clout.NewKey())

		workflows[workflow.Name] = v

		setVertexMetadata(v, "workflow")
		v.Properties["name"] = workflow.Name
		v.Properties["description"] = workflow.Description
	}

	// Workflow steps
	for name, workflow := range Workflows {
		v := workflows[name]

		steps := make(map[string]*clout.Vertex)

		for _, step := range workflow.Steps {
			sv := cloutP.NewVertex(clout.NewKey())

			steps[step.Name] = sv

			setVertexMetadata(sv, "workflowStep")
			sv.Properties["name"] = step.Name

			e := v.NewEdgeTo(sv)
			setVertexMetadata(e, "workflowStep")

			if step.TargetNodeTemplateOrGroupName != "" {
				nv := nodeTemplates[step.TargetNodeTemplateOrGroupName]
				e = sv.NewEdgeTo(nv)
				setVertexMetadata(e, "nodeTemplateTarget")
			} else {
				// This would happen only if there was a parsing error
				continue
			}

			// Workflow activities
			for sequence, activity := range step.Activities {
				av := cloutP.NewVertex(clout.NewKey())

				e = sv.NewEdgeTo(av)
				setVertexMetadata(e, "workflowActivity")
				e.Properties["sequence"] = sequence

				setVertexMetadata(av, "workflowActivity")
				if activity.DelegateWorkflow != nil {
					wv := workflows[activity.DelegateWorkflow.Name]
					e = av.NewEdgeTo(wv)
					setVertexMetadata(e, "delegateWorflow")
				} else if activity.InlineWorkflow != nil {
					wv := workflows[activity.InlineWorkflow.Name]
					e = av.NewEdgeTo(wv)
					setVertexMetadata(e, "inlineWorflow")
				} else if activity.CallOperation != "" {
					m := make(ard.Map)
					s := strings.Split(activity.CallOperation, ".")
					m["interface"] = s[0]
					m["operation"] = s[1]
					av.Properties["callOperation"] = m
				}
			}
		}

		// setup onSuccess and onFailure steps
		for _, step := range workflow.Steps {
			sv := steps[step.Name]

			for _, next := range step.OnSuccessSteps {
				nsv := steps[next]
				e := sv.NewEdgeTo(nsv)
				setVertexMetadata(e, "onSuccess")
			}

			for _, next := range step.OnFailureSteps {
				nsv := steps[next]
				e := sv.NewEdgeTo(nsv)
				setVertexMetadata(e, "onFailure")
			}
		}
	}
	return nil
}

func getInterfacesOperationsOfVertex(propertiesMap map[string]interface{}) []string {
	interfaces := propertiesMap["interfaces"].(map[string]interface{})
	standard := interfaces["Standard"].(map[string]interface{})
	operations := standard["operations"].(map[string]interface{})
	keys := make([]string, 0, len(operations))
	for key := range operations {
		keys = append(keys, key)
	}
	return keys
}

func isVertexNodeTemplate(vertex *clout.Vertex) bool {
	isVertexNodeTemplate := false
	vertexMetadata := vertex.Metadata
	if vertexMetadata != nil {
		metaDataName := vertexMetadata["puccini-tosca"].(map[string]interface{})
		kindName := metaDataName["kind"].(interface{})
		if kindName.(string) == "nodeTemplate" {
			isVertexNodeTemplate = true
		}
	}
	return isVertexNodeTemplate
}

func setVertexMetadata(entity clout.Entity, kind string) {
	metadata := make(ard.Map)
	metadata["version"] = version
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

// Workflows ...
type Workflows struct {
	WorkflowDefinitions WorkflowDefinitions `json:"workflows" yaml:"workflows"`
}

// NewWorkflows ...
func NewWorkflows() *Workflows {
	return &Workflows{
		WorkflowDefinitions: make(WorkflowDefinitions, 0),
	}
}

// WorkflowDefinition ...
type WorkflowDefinition struct {
	Name        string                  `json:"name" yaml:"name"`
	Description string                  `json:"description" yaml:"description"`
	Steps       WorkflowStepDefinitions `json:"steps" yaml:"steps"`
}

// NewWorkflowDefinition ...
func NewWorkflowDefinition() *WorkflowDefinition {
	return &WorkflowDefinition{
		Steps: make(WorkflowStepDefinitions),
	}
}

// WorkflowDefinitions ...
type WorkflowDefinitions map[string]*WorkflowDefinition

// WorkflowStepDefinition ...
type WorkflowStepDefinition struct {
	Name string `json:"name" yaml:"name"`

	TargetNodeTemplateOrGroupName string                      `json:"target" yaml:"target"`
	TargetNodeRequirementName     string                      `json:"target_relationship" yaml:"target_relationship"`
	OperationHost                 string                      `json:"operation_host" yaml:"operation_host"`
	TargetGroup                   string                      `json:"target_group" yaml:"target_group"`
	Activities                    WorkflowActivityDefinitions `json:"activities" yaml:"activities"`
	OnSuccessSteps                []string                    `json:"on_success" yaml:"on_success"`
	OnFailureSteps                []string                    `json:"on_failure" yaml:"on_failure"`
}

// NewWorkflowStepDefinition ...
func NewWorkflowStepDefinition() *WorkflowStepDefinition {
	return &WorkflowStepDefinition{
		Activities: make(WorkflowActivityDefinitions, 0),
	}
}

// WorkflowStepDefinitions ...
type WorkflowStepDefinitions map[string]*WorkflowStepDefinition

// WorkflowActivityDefinition ...
type WorkflowActivityDefinition struct {
	Step             string              `json:"-" yaml:"-"`
	DelegateWorkflow *WorkflowDefinition `json:"-" yaml:"-"`
	InlineWorkflow   *WorkflowDefinition `json:"-" yaml:"-"`
	CallOperation    string              `json:"call_operation" yaml:"call_operation"`
}

// NewWorkflowActivityDefinition ...
func NewWorkflowActivityDefinition() *WorkflowActivityDefinition {
	return &WorkflowActivityDefinition{}
}

// WorkflowActivityDefinitions ...
type WorkflowActivityDefinitions []*WorkflowActivityDefinition
