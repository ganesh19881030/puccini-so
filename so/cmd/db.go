package cmd

import (
	"context"
	"encoding/json"

	//"fmt"
	// "io/ioutil"
	//"github.com/tliron/puccini/url"

	//"log"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"

	//"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/database"
	//"github.com/tliron/puccini/tosca/parser"
	"google.golang.org/grpc"
)

var VERSION string

func createCloutOutput(dburl string, name string) (*clout.Clout, string) {

	//conn, err := grpc.Dial("localhost:9082", grpc.WithInsecure())
	conn, err := grpc.Dial(dburl, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	/*const q = `
	{
		all(func: has(<clout:vertex>)){
			expand(_all_) {
				expand(_all_) {
					expand(_all_)
				}
			}

		}
	}*/

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
	//resp, err := txn.Query(context.Background(), q)
	resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})
	if err != nil {
		//log.Fatal(err)
		return nil, ""
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}

	cloutOutput, uid := createClout(result)

	// Write into a file
	file, _ := json.MarshalIndent(cloutOutput, "", "  ")
	//file, _ := yaml.Marshal(cloutOutput)

	_ = ioutil.WriteFile("workflows_dgraph.json", file, 0644)

	return cloutOutput, uid

}

func createClout(result map[string]interface{}) (*clout.Clout, string) {
	cloutOutput := clout.NewClout()
	queryData := result["all"].([]interface{})

	if len(queryData) == 0 {
		log.Fatal("No results retrieved from the database")
	}
	cloutMap := queryData[0].(map[string]interface{})

	uid := cloutMap["uid"].(string)

	timestamp, err := common.Timestamp()
	if err != nil {
		log.Fatal(err)
	}

	metadata := make(ard.Map)

	grammarVersion := cloutMap["clout:grammarversion"].(string)
	internalImport := cloutMap["clout:import"].(string)
	scriptNamespace := database.CreateScriptNamespace(grammarVersion, internalImport)
	for name, jsEntry := range scriptNamespace {
		sourceCode, err := jsEntry.GetSourceCode()
		if err != nil {
			log.Fatal(err)
		}
		if err = js.SetMapNested(metadata, name, sourceCode); err != nil {
			log.Fatal(err)
		}
	}
	cloutOutput.Metadata["puccini-js"] = metadata

	metadata = make(ard.Map)
	VERSION = cloutMap["clout:version"].(string)
	metadata["version"] = VERSION
	metadata["history"] = []string{timestamp}
	cloutOutput.Metadata["puccini-tosca"] = metadata

	cloutProperties := cloutMap["clout:properties"].(string)

	var cloutProps map[string]interface{}
	if err := json.Unmarshal([]byte(cloutProperties), &cloutProps); err != nil {
		log.Fatal(err)
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
	nodeTemplates := addNodeTemplates(vertexList, cloutOutput)

	// Add Edges
	addEdges(vertexList, nodeTemplates)

	// Add Groups
	groups := addGroups(vertexList, nodeTemplates, cloutOutput)

	// Add WorkflowActivity
	workflowActivities := addWorkflowActivities(vertexList, cloutOutput)

	// Add WorkFlowSteps
	workflowSteps := addWorkflowSteps(vertexList, nodeTemplates, workflowActivities, cloutOutput)

	// Add WorkFlows
	addWorkflows(vertexList, workflowSteps, cloutOutput)

	// Add WorkflowActivity
	operations := addOperations(vertexList, cloutOutput)

	addPolicies(vertexList, nodeTemplates, groups, operations, cloutOutput)

	return cloutOutput, uid
}

func addNodeTemplates(vertexList []interface{}, cloutOutput *clout.Clout) map[string]*clout.Vertex {
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

				setMetadata(v, "nodeTemplate", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"] = getPropMap(templateMap["tosca:types"])
				v.Properties["directives"] = getPropStringList(templateMap["tosca:directives"])
				v.Properties["properties"] = getPropMap(templateMap["tosca:properties"])
				v.Properties["attributes"] = getPropMap(templateMap["tosca:attributes"])
				v.Properties["requirements"] = getPropList(templateMap["tosca:requirements"])
				//v.Properties["capabilities"] = templateMap["capabilities"]
				v.Properties["interfaces"] = getPropMap(templateMap["tosca:interfaces"])
				v.Properties["artifacts"] = getPropMap(templateMap["tosca:artifacts"])
				capMap := make(ard.Map)

				//Capabilities
				capabilityList := templateMap["capabilities"].([]interface{})
				for _, cap := range capabilityList {
					capability := cap.(map[string]interface{})
					c := clout.NewCapability()
					key := capability["tosca:key"].(string)
					c.Attributes = getPropMap(capability["tosca:attributes"])
					c.Description = capability["tosca:description"].(string)
					c.MaxRelationshipCount = capability["tosca:maxRelationshipCount"].(float64)
					c.MinRelationshipCount = capability["tosca:minRelationshipCount"].(float64)
					c.Properties = getPropMap(capability["tosca:properties"])
					c.Types = getPropMap(capability["tosca:types"])

					capMap[key] = c
				}
				v.Properties["capabilities"] = capMap

			}
		}
	}
	return nodeTemplates

}

func addEdges(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex) {
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
						setMetadata(edgeOut, "relationship", VERSION)
						edgeOut.Properties["attributes"] = getPropMap(edgeMap["clout:edge|tosca:attributes"])
						edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
						edgeOut.Properties["description"] = edgeMap["clout:edge|tosca:description"]
						edgeOut.Properties["interfaces"] = getPropMap(edgeMap["clout:edge|tosca:interfaces"])
						edgeOut.Properties["name"] = edgeMap["clout:edge|tosca:name"]
						edgeOut.Properties["properties"] = getPropMap(edgeMap["clout:edge|tosca:properties"])
						edgeOut.Properties["types"] = getPropMap(edgeMap["clout:edge|tosca:types"])
						edgeOut.TargetID = vv.ID
						edgeOuts = append(edgeOuts, edgeOut)
					}

				}
				v.EdgesOut = edgeOuts
			}
		}
	}
}

func addGroups(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, cloutOutput *clout.Clout) map[string]*clout.Vertex {
	groups := make(map[string]*clout.Vertex)
	for _, node := range vertexList {

		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "group" {
				//v := cloutOutput.NewVertex(clout.NewKey())
				v := cloutOutput.NewVertex(templateMap["tosca:vertexId"].(string))
				//groups[templateMap["tosca:name"].(string)] = v
				groups[templateMap["tosca:vertexId"].(string)] = v

				setMetadata(v, "group", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"] = getPropMap(templateMap["tosca:types"])
				v.Properties["properties"] = getPropMap(templateMap["tosca:properties"])
				v.Properties["interfaces"] = getPropMap(templateMap["tosca:interfaces"])

				members := templateMap["clout:edge"].([]interface{})
				for _, nodeTemplate := range members {
					nodeMap := nodeTemplate.(map[string]interface{})
					//nv := nodeTemplates[nodeMap["tosca:name"].(string)]
					nv := nodeTemplates[nodeMap["tosca:vertexId"].(string)]
					e := v.NewEdgeTo(nv)

					setMetadata(e, "member", VERSION)
				}
			}
		}
	}
	return groups

}

func addWorkflows(vertexList []interface{}, workflowSteps map[string]*clout.Vertex, cloutOutput *clout.Clout) {
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

				setMetadata(v, "workflow", VERSION)
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
							setMetadata(edgeOut, "workflowStep", VERSION)
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
	//return workflows
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

				setMetadata(v, "workflowStep", VERSION)
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
					setMetadata(edgeOut, edgeEntity, VERSION)
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

func addWorkflowActivities(vertexList []interface{}, cloutOutput *clout.Clout) map[string]*clout.Vertex {
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

				setMetadata(v, "workflowActivity", VERSION)
				if templateMap["tosca:setNodeState"] != nil {
					v.Properties["setNodeState"] = templateMap["tosca:setNodeState"]
				}
				if templateMap["tosca:callOperation"] != nil {
					v.Properties["callOperation"] = getPropMap(templateMap["tosca:callOperation"])
					//v.Properties["callOperation"] = templateMap["tosca:callOperation"]
				}

			}
		}
	}
	//addWorkflowStepEdges(workflowNodes, workflowSteps, nodeTemplates)
	return workflowActivities
}

func addOperations(vertexList []interface{}, cloutOutput *clout.Clout) map[string]*clout.Vertex {
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

				setMetadata(v, "operation", VERSION)
				v.Properties["dependencies"] = getPropList(templateMap["tosca:dependencies"])
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["implementation"] = templateMap["tosca:implementation"]
				v.Properties["inputs"] = getPropMap(templateMap["tosca:inputs"])

			}
		}
	}
	return operations
}

func addPolicies(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, groups map[string]*clout.Vertex, operations map[string]*clout.Vertex, cloutOutput *clout.Clout) map[string]*clout.Vertex {
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

				setMetadata(v, "policy", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"] = getPropMap(templateMap["tosca:types"])
				v.Properties["properties"] = getPropMap(templateMap["tosca:properties"])
				cloutEdge := templateMap["clout:edge"]
				if cloutEdge != nil {
					edgeList := cloutEdge.([]interface{})
					for _, edge := range edgeList {
						edgeMap := edge.(map[string]interface{})
						if edgeMap["tosca:entity"] == "nodeTemplate" {
							//vv := nodeTemplates[edgeMap["tosca:name"].(string)]
							vv := nodeTemplates[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, "nodeTemplateTarget", VERSION)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
							//edgeOut.Properties["capability"] = edgeMap["clout:edge|tosca:capability"]
						} else if edgeMap["tosca:entity"] == "group" {
							//vv := groups[edgeMap["tosca:name"].(string)]
							vv := groups[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, "groupTarget", VERSION)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
						} else if edgeMap["tosca:entity"] == "operation" {
							vv := operations[edgeMap["tosca:vertexId"].(string)]
							edgeOut := v.NewEdgeTo(vv)
							setMetadata(edgeOut, edgeMap["clout:edge|tosca:entity"].(string), VERSION)
							edgeOut.TargetID = vv.ID
							edgeOuts = append(edgeOuts, edgeOut)
						}
					}
				}
				v.EdgesOut = edgeOuts
			}
		}
	}
	return policies
}

func setMetadata(entity clout.Entity, kind string, version string) {
	metadata := make(ard.Map)
	metadata["version"] = version
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

func getPropMap(prop interface{}) ard.Map {
	props := make(ard.Map)
	if prop != nil {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Fatal(err)
		}
	}
	return props
}

func getPropStringList(prop interface{}) []string {
	props := make([]string, 0)
	if prop != nil {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Fatal(err)
		}
	}
	return props
}

func getPropList(prop interface{}) []ard.Map {
	props := make([]ard.Map, 0)
	if prop != nil {
		propString := prop.(string)
		if err := json.Unmarshal([]byte(propString), &props); err != nil {
			log.Fatal(err)
		}
	}
	return props
}
