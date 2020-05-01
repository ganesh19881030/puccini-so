package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/tosca/database"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

var TemplateVersion string

func isCloutPresent(dgt *dgraph.DgraphTemplate, name string) (bool, error) {

	var result ard.Map

	// Query the clout vertex by name
	// TODO: not adequate - need to have a better query with more criteria
	const paramquery = `{all(func: has(<clout:grammarversion>)) @filter (eq (<clout:name>,"%s")){
		uid
		<clout:name>
		<clout:version>
		<clout:grammarversion>
	  }
	}`

	query := fmt.Sprintf(paramquery, name)

	resp, err := dgt.ExecQuery(query)
	found := false
	//resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})
	if err == nil {
		if err := json.Unmarshal(resp.GetJson(), &result); err == nil {
			if aresp, ok := result["all"]; ok {
				if arr, ok := aresp.([]interface{}); ok {
					if len(arr) > 0 {
						found = true
					}
				}
			}
		}
	}

	return found, err
}

func readClout(dburl string, name string) (*ard.Map, error) {

	var result ard.Map

	/*
		conn, err := grpc.Dial(dburl, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()
		dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
		txn := dgraphClient.NewTxn()
		ctx := context.Background()
		defer txn.Discard(ctx)


		// Query the clout vertex by name
		const q = `query all($name: string) {
			all(func: eq(<clout:name>, $name)) {
				uid
				expand(_all_) {
					expand(_all_) {
					  expand(_all_)
					}
				  }
			  }
		  }`

		resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})

	*/
	const paramquery = `{
		all(func: eq(<clout:name>, "%s")) {
			uid
			expand(_all_) {
				expand(_all_) {
				  expand(_all_)
				}
			  }
		  }
	  }`

	dgt, err := fetchDbTemplate()
	common.FailOnError(err)
	defer dgt.Close()
	query := fmt.Sprintf(paramquery, name)

	resp, err := dgt.ExecQuery(query)

	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	err = json.Unmarshal(resp.GetJson(), &result)

	return &result, err
}
func createCloutOutput(dburl string, name string) (*clout.Clout, string, error) {

	result, err := readClout(dburl, name)

	if err != nil {
		log.Errorf(err.Error())
		return nil, "", err

	}

	cloutOutput, uid, err := createClout(*result)

	if err == nil {
		// Write into a file
		file, _ := json.MarshalIndent(cloutOutput, "", "  ")
		//file, _ := yaml.Marshal(cloutOutput)

		_ = ioutil.WriteFile("fw1_csar_dgraph.json", file, 0644)
	}

	return cloutOutput, uid, err

}

func createClout(result map[string]interface{}) (*clout.Clout, string, error) {

	cloutOutput := clout.NewClout()
	queryData := result["all"].([]interface{})

	if len(queryData) == 0 {
		err := errors.New("No results retrieved from the database")
		log.Errorf(err.Error())
		return nil, "", err
	}
	cloutMap := queryData[0].(map[string]interface{})

	uid := cloutMap["uid"].(string)

	timestamp, err := common.Timestamp()
	if err != nil {
		log.Errorf(err.Error())
		return nil, "", err
	}

	metadata := make(ard.Map)

	grammarVersion := cloutMap["clout:grammarversion"].(string)
	internalImport := cloutMap["clout:import"].(string)
	scriptNamespace := database.CreateScriptNamespace(grammarVersion, internalImport)
	for name, jsEntry := range scriptNamespace {
		sourceCode, err := jsEntry.GetSourceCode()
		if err != nil {
			log.Errorf(err.Error())
			return nil, "", err
		}
		if err = js.SetMapNested(metadata, name, sourceCode); err != nil {
			log.Errorf(err.Error())
			return nil, "", err
		}
	}
	cloutOutput.Metadata["puccini-js"] = metadata

	metadata = make(ard.Map)
	TemplateVersion = cloutMap["clout:version"].(string)
	metadata["version"] = TemplateVersion
	metadata["history"] = []string{timestamp}
	cloutOutput.Metadata["puccini-tosca"] = metadata

	cloutProperties := cloutMap["clout:properties"].(string)

	var cloutProps map[string]interface{}
	if err := json.Unmarshal([]byte(cloutProperties), &cloutProps); err != nil {
		log.Errorf(err.Error())
		return nil, "", err
	}

	tosca := make(ard.Map)
	tosca["description"] = cloutProps["description"]
	tosca["metadata"] = cloutProps["metadata"]
	tosca["inputs"] = cloutProps["inputs"]
	tosca["outputs"] = cloutProps["outputs"]
	cloutOutput.Properties["tosca"] = tosca

	cloutVertex := cloutMap["clout:vertex"]
	vertexList := cloutVertex.([]interface{})

	//Add nodeTemplates
	nodeTemplates, err := addNodeTemplates(vertexList, cloutOutput)
	if err != nil {
		return nil, "", err
	}

	// Add Edges
	addEdges(vertexList, nodeTemplates)

	// Add Groups
	groups, err := addGroups(vertexList, nodeTemplates, cloutOutput)
	if err != nil {
		return nil, "", err
	}

	// Add WorkflowActivity
	workflowActivities, err := addWorkflowActivities(vertexList, cloutOutput)
	if err != nil {
		return nil, "", err
	}

	// Add WorkFlowSteps
	workflowSteps := addWorkflowSteps(vertexList, nodeTemplates, workflowActivities, cloutOutput)

	// Add WorkFlows
	addWorkflows(vertexList, workflowSteps, cloutOutput)

	// Add Operations
	operations, err := addOperations(vertexList, cloutOutput)
	if err != nil {
		return nil, "", err
	}

	// Add Policy Triggers
	triggers, err := addPolicyTriggers(vertexList, cloutOutput)
	if err != nil {
		return nil, "", err
	}

	addPolicies(vertexList, nodeTemplates, groups, triggers, cloutOutput)

	addSubstitutions(vertexList, nodeTemplates, groups, operations, cloutOutput)

	return cloutOutput, uid, nil
}

func addNodeTemplates(vertexList []interface{}, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error
	// Node templates
	nodeTemplates := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "nodeTemplate" {
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))
				//nodeTemplates[templateMap["tosca:name"].(string)] = v
				nodeTemplates[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "nodeTemplate", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"], err = getPropMap(templateMap["tosca:types"])
				if err != nil {
					return nil, err
				}
				v.Properties["directives"], err = getPropList(templateMap["tosca:directives"])
				if err != nil {
					return nil, err
				}
				v.Properties["properties"], err = getPropMap(templateMap["tosca:properties"])
				if err != nil {
					return nil, err
				}
				v.Properties["attributes"], err = getPropMap(templateMap["tosca:attributes"])
				if err != nil {
					return nil, err
				}
				v.Properties["requirements"], err = getPropList(templateMap["tosca:requirements"])
				if err != nil {
					return nil, err
				}
				//v.Properties["capabilities"] = templateMap["capabilities"]
				v.Properties["interfaces"], err = getPropMap(templateMap["tosca:interfaces"])
				if err != nil {
					return nil, err
				}
				v.Properties["artifacts"], err = getPropMap(templateMap["tosca:artifacts"])
				if err != nil {
					return nil, err
				}
				capMap := make(ard.Map)

				//Capabilities
				capabilityList := templateMap["tosca:capabilities"].([]interface{})
				for _, cap := range capabilityList {
					capability := cap.(map[string]interface{})
					c := clout.NewCapability()
					key := capability["tosca:key"].(string)
					c.Attributes, err = getPropMap(capability["tosca:attributes"])
					if err != nil {
						return nil, err
					}
					c.Description = capability["tosca:description"].(string)
					c.MaxRelationshipCount = capability["tosca:maxRelationshipCount"].(float64)
					c.MinRelationshipCount = capability["tosca:minRelationshipCount"].(float64)
					c.Properties, err = getPropMap(capability["tosca:properties"])
					if err != nil {
						return nil, err
					}
					c.Types, err = getPropMap(capability["tosca:types"])
					if err != nil {
						return nil, err
					}
					capMap[key] = c
				}
				v.Properties["capabilities"] = capMap

			}
		}
	}
	return nodeTemplates, err

}

func addEdges(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex) error {
	var err error

	for _, subNode := range vertexList {

		templateMap := subNode.(map[string]interface{})
		edgeOuts := make(clout.Edges, 0)
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "nodeTemplate" {
				//v := nodeTemplates[templateMap["tosca:name"].(string)]
				v := nodeTemplates[templateMap["tosca:vertexId"].(string)]

				// Edges
				cloutEdge := templateMap["clout:edge"]
				if cloutEdge != nil {
					edgeList := cloutEdge.([]interface{})
					for _, edge := range edgeList {
						edgeMap := edge.(map[string]interface{})
						//vv := nodeTemplates[edgeMap["tosca:name"].(string)]
						vv := nodeTemplates[edgeMap["tosca:vertexId"].(string)]
						edgeOut := v.NewEdgeTo(vv)
						setMetadata(edgeOut, "relationship", TemplateVersion)
						edgeOut.Properties["attributes"], err = getPropMap(edgeMap["clout:edge|tosca:attributes"])
						if err != nil {
							return err
						}
						edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
						edgeOut.Properties["description"] = edgeMap["clout:edge|tosca:description"]
						edgeOut.Properties["interfaces"], err = getPropMap(edgeMap["clout:edge|tosca:interfaces"])
						if err != nil {
							return err
						}
						edgeOut.Properties["name"] = edgeMap["clout:edge|tosca:name"]
						edgeOut.Properties["properties"], err = getPropMap(edgeMap["clout:edge|tosca:properties"])
						if err != nil {
							return err
						}
						edgeOut.Properties["types"], err = getPropMap(edgeMap["clout:edge|tosca:types"])
						if err != nil {
							return err
						}
						edgeOut.TargetID = vv.ID
						edgeOuts = append(edgeOuts, edgeOut)
					}

				}
				v.EdgesOut = edgeOuts
			}
		}
	}
	return err
}

func addGroups(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error
	groups := make(map[string]*clout.Vertex)
	for _, node := range vertexList {

		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "group" {
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))
				//groups[templateMap["tosca:name"].(string)] = v
				groups[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "group", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"], err = getPropMap(templateMap["tosca:types"])
				if err != nil {
					return nil, err
				}
				v.Properties["properties"], err = getPropMap(templateMap["tosca:properties"])
				if err != nil {
					return nil, err
				}
				v.Properties["interfaces"], err = getPropMap(templateMap["tosca:interfaces"])
				if err != nil {
					return nil, err
				}

				members := templateMap["clout:edge"].([]interface{})
				for _, nodeTemplate := range members {
					nodeMap := nodeTemplate.(map[string]interface{})
					//nv := nodeTemplates[nodeMap["tosca:name"].(string)]
					nv := nodeTemplates[nodeMap["tosca:vertexId"].(string)]
					e := v.NewEdgeTo(nv)

					setMetadata(e, "member", TemplateVersion)
				}
			}
		}
	}
	return groups, err

}

func addWorkflows(vertexList []interface{}, workflowSteps map[string]*clout.Vertex, cloutOutput *clout.Clout) error {
	var err error
	//workflows := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())
		edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "workflow" {
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))
				//workflows[templateMap["tosca:name"].(string)] = v
				//workflows[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "workflow", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				cloutEdge := templateMap["clout:edge"]
				if cloutEdge != nil {
					edgeList := cloutEdge.([]interface{})
					for _, edge := range edgeList {
						edgeMap := edge.(map[string]interface{})
						if edgeMap["tosca:entity"] == "workflowStep" {
							//vv := workflowSteps[edgeMap["tosca:name"].(string)]
							vv := workflowSteps[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, "workflowStep", TemplateVersion)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
							//edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
						}
					}
				}
				v.EdgesOut = edgeOuts
			}
		}
	}
	return err
}

func addWorkflowSteps(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex,
	workflowActivities map[string]*clout.Vertex, cloutOutput *clout.Clout) map[string]*clout.Vertex {

	workflowSteps := make(map[string]*clout.Vertex)
	workflowNodes := make([]interface{}, 0)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())
		//edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "workflowStep" {
				//groups := make(map[string]*clout.Vertex)
				workflowNodes = append(workflowNodes, node)
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))

				//workflowSteps[templateMap["tosca:name"].(string)] = v
				workflowSteps[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "workflowStep", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				//v.Properties["firstStep"] = templateMap["tosca:firstStep"]
			}
		}
	}
	addWorkflowStepEdges(workflowNodes, workflowSteps, nodeTemplates, workflowActivities, cloutOutput)
	return workflowSteps
}

func addWorkflowStepEdges(nodes []interface{}, workflowSteps map[string]*clout.Vertex,
	nodeTemplates map[string]*clout.Vertex, workflowActivities map[string]*clout.Vertex, cloutOutput *clout.Clout) {

	var edgeOut *clout.Edge
	var vv *clout.Vertex
	for _, node := range nodes {
		//v := cloutOutput.NewVertex(clout.NewKey())
		edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		//if entity, ok := templateMap["tosca:entity"]; ok {
		//v := workflowSteps[templateMap["tosca:name"].(string)]
		v := workflowSteps[templateMap["tosca:vertexId"].(string)]

		cloutEdge := templateMap["clout:edge"]
		if cloutEdge != nil {
			edgeList := cloutEdge.([]interface{})

			for _, edge := range edgeList {
				edgeMap := edge.(map[string]interface{})
				vv = nil //initialize the vertex
				if edgeMap["tosca:entity"] == "nodeTemplate" {
					//vv = nodeTemplates[edgeMap["tosca:name"].(string)]
					vv = nodeTemplates[edgeMap["tosca:vertexId"].(string)]
				} else if edgeMap["tosca:entity"] == "workflowStep" {
					//vv = workflowSteps[edgeMap["tosca:name"].(string)]
					vv = workflowSteps[edgeMap["tosca:vertexId"].(string)]
				} else if edgeMap["tosca:entity"] == "workflowActivity" {
					vv = workflowActivities[edgeMap["tosca:vertexId"].(string)]
				}

				if vv != nil {
					edgeEntity := edgeMap["clout:edge|tosca:entity"].(string)
					edgeOut = v.NewEdgeTo(vv)
					setMetadata(edgeOut, edgeEntity, TemplateVersion)
					if edgeMap["clout:edge|tosca:sequence"] != nil {
						edgeOut.Properties["sequence"] = edgeMap["clout:edge|tosca:sequence"]
					}
					edgeOut.TargetID = vv.ID
					edgeOuts = append(edgeOuts, edgeOut)
				}

			}
		}
		v.EdgesOut = edgeOuts

	}

}

func addWorkflowActivities(vertexList []interface{}, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error

	workflowActivities := make(map[string]*clout.Vertex)
	workflowActivityNodes := make([]interface{}, 0)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())
		//edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "workflowActivity" {
				//groups := make(map[string]*clout.Vertex)
				workflowActivityNodes = append(workflowActivityNodes, node)
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))

				workflowActivities[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "workflowActivity", TemplateVersion)
				if templateMap["tosca:setNodeState"] != nil {
					v.Properties["setNodeState"] = templateMap["tosca:setNodeState"]
				}
				if templateMap["tosca:callOperation"] != nil {
					v.Properties["callOperation"], err = getPropMap(templateMap["tosca:callOperation"])
					if err != nil {
						return nil, err
					}

					//v.Properties["callOperation"] = templateMap["tosca:callOperation"]
				}

			}
		}
	}
	//addWorkflowStepEdges(workflowNodes, workflowSteps, nodeTemplates)
	return workflowActivities, err
}

func addOperations(vertexList []interface{}, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error

	operations := make(map[string]*clout.Vertex)
	operationNodes := make([]interface{}, 0)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())
		//edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "operation" {
				//groups := make(map[string]*clout.Vertex)
				operationNodes = append(operationNodes, node)
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))

				operations[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "operation", TemplateVersion)
				v.Properties["dependencies"], err = getPropList(templateMap["tosca:dependencies"])
				if err != nil {
					return nil, err
				}
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["implementation"] = templateMap["tosca:implementation"]
				v.Properties["inputs"], err = getPropMap(templateMap["tosca:inputs"])
				if err != nil {
					return nil, err
				}

			}
		}
	}
	return operations, err
}

func addPolicyTriggers(vertexList []interface{}, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error

	policyTriggers := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "policyTrigger" {
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))
				policyTriggers[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "policyTrigger", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["event_type"] = templateMap["tosca:event_type"]

				if templateMap["tosca:condition"] != nil {
					v.Properties["condition"], err = getPropMap(templateMap["tosca:condition"])
					if err != nil {
						return nil, err
					}
				}
				if templateMap["tosca:action"] != nil {
					v.Properties["action"], err = getPropMap(templateMap["tosca:action"])
					if err != nil {
						return nil, err
					}
				}
				if templateMap["tosca:operation"] != nil {
					v.Properties["operation"], err = getPropMap(templateMap["tosca:operation"])
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return policyTriggers, err
}

func addPolicies(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, groups map[string]*clout.Vertex,
	policyTriggers map[string]*clout.Vertex, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error

	policies := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())
		edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "policy" {
				//groups := make(map[string]*clout.Vertex)
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))

				//policies[templateMap["tosca:name"].(string)] = v
				policies[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "policy", TemplateVersion)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"], err = getPropMap(templateMap["tosca:types"])
				if err != nil {
					return nil, err
				}
				v.Properties["properties"], err = getPropMap(templateMap["tosca:properties"])
				if err != nil {
					return nil, err
				}
				cloutEdge := templateMap["clout:edge"]
				if cloutEdge != nil {
					edgeList := cloutEdge.([]interface{})
					for _, edge := range edgeList {
						edgeMap := edge.(map[string]interface{})
						if edgeMap["tosca:entity"] == "nodeTemplate" {
							//vv := nodeTemplates[edgeMap["tosca:name"].(string)]
							vv := nodeTemplates[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, "nodeTemplateTarget", TemplateVersion)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
							//edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
						} else if edgeMap["tosca:entity"] == "group" {
							//vv := groups[edgeMap["tosca:name"].(string)]
							vv := groups[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, "groupTarget", TemplateVersion)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
						} else if edgeMap["tosca:entity"] == "policyTrigger" {
							vv := policyTriggers[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, edgeMap["clout:edge|tosca:entity"].(string), TemplateVersion)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
						}
					}
				}
				v.EdgesOut = edgeOuts
			}
		}
	}
	return policies, err
}

func addSubstitutions(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, groups map[string]*clout.Vertex, operations map[string]*clout.Vertex, cloutOutput *clout.Clout) (map[string]*clout.Vertex, error) {
	var err error

	subs := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		i := 0
		edgeOuts := make(clout.Edges, 0)
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "substitution" {
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))

				subs[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "substitution", TemplateVersion)
				v.Properties["substitutionFilter"], err = getPropList(templateMap["tosca:substitutionFilter"])
				if err != nil {
					return nil, err
				}
				v.Properties["type"] = templateMap["tosca:type"]
				v.Properties["typeMetadata"], err = getPropMap(templateMap["tosca:typeMetadata"])
				if err != nil {
					return nil, err
				}
				v.Properties["properties"], err = getPropMap(templateMap["tosca:properties"])
				if err != nil {
					return nil, err
				}
				cloutEdge := templateMap["clout:edge"]

				for ok := true; ok; ok = cloutEdge != nil {
					var prefix string
					if i != 0 {
						prefix = "clout:edge" + strconv.Itoa(i)
					} else {
						prefix = "clout:edge"
					}

					if cloutEdge != nil {
						edgeList := cloutEdge.([]interface{})
						for _, edge := range edgeList {
							edgeMap := edge.(map[string]interface{})
							entity := edgeMap[prefix+"|tosca:entity"].(string)
							vv := nodeTemplates[edgeMap["clout:edge|tosca:vertexId"].(string)]
							if vv != nil {
								//if edgeMap["tosca:entity"] == "nodeTemplate" {
								//vv := nodeTemplates[edgeMap["tosca:vertexId"].(string)]
								edgeOut := v.NewEdgeTo(vv)
								setMetadata(edgeOut, entity, TemplateVersion)
								edgeOut.TargetID = vv.ID
								edgeOuts = append(edgeOuts, edgeOut)
								if edgeMap[prefix+"|tosca:requirement"] != nil {
									edgeOut.Properties["requirement"] = edgeMap[prefix+"|tosca:requirement"]
								}
								if edgeMap[prefix+"|tosca:requirementName"] != nil {
									edgeOut.Properties["requirementName"] = edgeMap[prefix+"|tosca:requirementName"]
								}
								if edgeMap[prefix+"|tosca:capability"] != nil {
									edgeOut.Properties["capability"] = edgeMap[prefix+"|tosca:capability"]
								}
								if edgeMap[prefix+"|tosca:capabilityName"] != nil {
									edgeOut.Properties["capabilityName"] = edgeMap[prefix+"|tosca:capabilityName"]
								}
								if edgeMap[prefix+"|tosca:interface"] != nil {
									edgeOut.Properties["interface"] = edgeMap[prefix+"|tosca:interface"]
								}
								//edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
							}
						}
					}
					i++
					cloutEdge = templateMap["clout:edge"+strconv.Itoa(i)]
				}

				v.EdgesOut = edgeOuts
			}
		}
	}
	return subs, err
}

func setMetadata(entity clout.Entity, kind string, version string) {
	metadata := make(ard.Map)
	metadata["version"] = version
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

func getPropMap(prop interface{}) (ard.Map, error) {
	props := make(ard.Map)
	if prop != nil {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Errorf(err.Error())
			return nil, err
		}
	}
	return props, nil
}

func getPropStringList(prop interface{}) ([]string, error) {
	props := make([]string, 0)
	if prop != nil && prop != "" {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Errorf(err.Error())
			return nil, err
		}
	}
	return props, nil
}

func getPropList(prop interface{}) ([]interface{}, error) {
	props := make([]interface{}, 0)
	if prop != nil && prop != "" {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Errorf(err.Error())
			return nil, err
		}
	}
	return props, nil
}
