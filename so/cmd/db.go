package cmd

import (
	"context"
	"encoding/json"

	//"log"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/grammars/tosca_v1_2"
	"google.golang.org/grpc"

	"github.com/tliron/puccini/js"
)

var VERSION string

func createCloutOutput(dburl string) *clout.Clout {

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

	// Query the name and description of all nodes
	const q = `
		{
			all(func: has(<clout:vertex>)){
				expand(_all_) {
					expand(_all_) {
						expand(_all_)
					}
				}
				 
			}
		}
	`
	resp, err := txn.Query(context.Background(), q)
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}

	cloutOutput := createClout(result)

	// Write into a file
	//file, _ := json.MarshalIndent(cloutOutput, "", " ")
	//file, _ := yaml.Marshal(cloutOutput)

	//_ = ioutil.WriteFile("test4.json", file, 0644)

	return cloutOutput

}

func createClout(result map[string]interface{}) *clout.Clout {
	cloutOutput := clout.NewClout()
	queryData := result["all"].([]interface{})
	cloutMap := queryData[0].(map[string]interface{})

	//toscaContext := tosca.NewContext(nil, nil)
	//vers := "tosca_simple_yaml_1_2"
	//toscaContext.Grammar = parser.Grammars[vers]
	//grammarVersion := cloutMap["clout:grammarversion"]
	//path := "github.com/tliron/puccini/tosca/grammars/" + grammarVersion
	//fmt.Println(grammarVersion)
	//grammarVersion.

	timestamp, err := common.Timestamp()
	if err != nil {
		log.Fatal(err)
	}

	metadata := make(ard.Map)

	//for name, jsEntry := range tosca_v1_2.Grammar.DefaultScriptNamespace {
	//parser.Grammars["tosca_simple_yaml_1_2"].DefaultScriptNamespace
	//var toscaContext *tosca.Context
	//toscaContext := tosca.NewContext(nil, nil)
	//toscaContext.ScriptNamespace.Merge(tosca_v1_2.DefaultScriptNamespace)
	//read := tosca_v1_2.Grammar["ServiceTemplate"]
	//data := read(toscaContext)
	//fmt.Println(data.(string))
	for name, jsEntry := range tosca_v1_2.DefaultScriptNamespace {
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
	addGroups(vertexList, nodeTemplates, cloutOutput)

	// Add WorkFlows
	//addWorkFlows()

	// Add WorkFlowSteps
	//addWorkFlowSteps()

	return cloutOutput
}

func addNodeTemplates(vertexList []interface{}, cloutOutput *clout.Clout) map[string]*clout.Vertex {
	// Node templates
	//var *nodeTemplates := templates
	nodeTemplates := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "nodeTemplate" {
				v := cloutOutput.NewVertex(clout.NewKey())
				nodeTemplates[templateMap["tosca:name"].(string)] = v

				SetMetadata(v, "nodeTemplate", VERSION)
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
				v := nodeTemplates[templateMap["tosca:name"].(string)]

				// Edges
				cloutEdge := templateMap["clout:edge"]
				if cloutEdge != nil {
					edgeList := cloutEdge.([]interface{})
					for _, edge := range edgeList {
						edgeMap := edge.(map[string]interface{})
						vv := nodeTemplates[edgeMap["clout:edge|tosca:name"].(string)]
						edgeOut := v.NewEdgeTo(vv)
						SetMetadata(edgeOut, "relationship", VERSION)
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

func addGroups(vertexList []interface{}, nodeTemplates map[string]*clout.Vertex, cloutOutput *clout.Clout) {
	groups := make(map[string]*clout.Vertex)
	for _, node := range vertexList {

		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "group" {
				v := cloutOutput.NewVertex(clout.NewKey())

				groups[templateMap["tosca:name"].(string)] = v

				SetMetadata(v, "group", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
				v.Properties["types"] = getPropMap(templateMap["tosca:types"])
				v.Properties["properties"] = getPropMap(templateMap["tosca:properties"])
				v.Properties["interfaces"] = getPropMap(templateMap["tosca:interfaces"])

				members := templateMap["clout:edge"].([]interface{})
				for _, nodeTemplate := range members {
					nodeMap := nodeTemplate.(map[string]interface{})
					nv := nodeTemplates[nodeMap["tosca:name"].(string)]
					e := v.NewEdgeTo(nv)

					SetMetadata(e, "member", VERSION)
				}
			}
		}
	}

}

/*func addWorkFlows() {
	workflows := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())

		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "workflow" {
				//groups := make(map[string]*clout.Vertex)
				v := cloutOutput.NewVertex(clout.NewKey())

				workflows[templateMap["tosca:name"].(string)] = v

				SetMetadata(v, "workflow", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
			}
		}
	}
}*/

/*func addWorkFlowSteps() {
	workflows := make(map[string]*clout.Vertex)
	for _, node := range vertexList {
		//v := cloutOutput.NewVertex(clout.NewKey())

		templateMap := node.(map[string]interface{})
		if entity, ok := templateMap["tosca:entity"]; ok {
			if entity == "workflow" {
				//groups := make(map[string]*clout.Vertex)
				v := cloutOutput.NewVertex(clout.NewKey())

				workflows[templateMap["tosca:name"].(string)] = v

				SetMetadata(v, "workflow", VERSION)
				v.Properties["name"] = templateMap["tosca:name"]
				v.Properties["description"] = templateMap["tosca:description"]
			}
		}
	}
}*/

func SetMetadata(entity clout.Entity, kind string, version string) {
	metadata := make(ard.Map)
	metadata["version"] = version
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

/*func getPropMap(propString string) ard.Map {
	var props ard.Map
	//fmt.Println(propString)
	//fmt.Println("##################################################################")
	if err := json.Unmarshal([]byte(propString), &props); err != nil {
		log.Fatal(err)
	}
	return props
}*/

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
