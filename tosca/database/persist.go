package database

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"encoding/json"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

// DgraphSet for Dgraph json data
type DgraphSet struct {
	Set []ard.Map `json:"set"`
}

// Persist description
func Persist(clout *clout.Clout, urlString string, grammarVersion string) error {

	// read configuration from a file
	socfg,err := common.GetSoConfig()
	if err != nil {
		common.FailOnError(err)
	}

	var printout = false
	//	timestamp, err := common.Timestamp()
	//	if err != nil {
	//		return  err
	//	}
	//	var metadata1 interface{} = clout_.Metadata["puccini-js"]
	//	metadata2 := clout_.Metadata["puccini-tosca"]
	//
	//	tosca := clout_.Properties["tosca"]

	var dgraphset = DgraphSet{}

	var vertexItems []ard.Map
	var cloutItem = make(ard.Map)

	//	nodeTemplates := make(map[string]*clout.Vertex)
	//jscripts := clout.Metadata["puccini-js"].(ard.Map)
	/*
		for key, script := range jscripts {

			fmt.Println("key: ", key)
			fmt.Println("script: ", script)

		}
	*/
	//cloutItem["scripts"] = jscripts
	/*
		data := clout.Metadata["puccini-tosca"].(ard.Map)
		for key, value := range data {

			fmt.Println("key: ", key)
			fmt.Println("value: ", value)

		}

		data = clout.Properties["tosca"].(ard.Map)
		mdata := data["metadata"].(ard.Map)
		for key, value := range mdata {

			fmt.Println("key: ", key)
			fmt.Println("value: ", value)

		}
	*/
	// Node templates
	for _, vertex := range clout.Vertexes {
		//		v := clout_.NewVertex(clout.NewKey())

		ind := vertex.ID
		vxItem := make(ard.Map)
		vxItem["uid"] = "_:clout.vertex." + ind
		vxItem["clout:edge"] = make([]*ard.Map, 0)

		if isToscaVertex(vertex, "nodeTemplate") {
			fillNodeTemplate(&vxItem, &vertex.Properties)
		} else if isToscaVertex(vertex, "group") {
			fillTosca(&vxItem, &vertex.Properties, "group", "")
		} else if isToscaVertex(vertex, "workflow") {
			fillTosca(&vxItem, &vertex.Properties, "workflow", "")
		} else if isToscaVertex(vertex, "policy") {
			fillTosca(&vxItem, &vertex.Properties, "policy", "")
		}

		//		var vertexItem string = "{uid: '_:clout.vertex.'" + ind + ", 'clout:edge': []}";

		for _, edge := range vertex.EdgesOut {
			//			var edgeItem = "{uid: '_:clout.vertex.'" + edge.Target.ID + "}"
			fillEdge(&vxItem, edge)
		}

		vertexItems = append(vertexItems, vxItem)

	}

	cloutItem["clout:vertex"] = vertexItems

	topologyName := extractTopologyName(urlString)
	cloutItem["clout:name"] = topologyName
	cloutItem["clout:version"] = clout.Version
	cloutItem["clout:grammarversion"] = grammarVersion
	props := clout.Properties["tosca"].(ard.Map)

	bytes, error := json.Marshal(props)
	if error == nil {
		//fmt.Println(" clout properties: \n", string(bytes))
		cloutItem["clout:properties"] = string(bytes)
	}

	//for key, value := range props {
	//	fmt.Println("key: ", key, "value: ", value)
	//}
	dgraphset.Set = append(dgraphset.Set, cloutItem)

	// write out the Dgraph data in JSON format
	if printout {
		err := format.WriteOrPrint(dgraphset, "json", true, "")
		common.FailOnError(err)
		fmt.Println("-")
		fmt.Println("---------------------------------------------------")
		fmt.Println("-")
	}

	// construct Dgraph url from configuration
	dburl := fmt.Sprintf("%s:%d", socfg.Dgraph.Host, socfg.Dgraph.Port)
	// save clout data into Dgraph
	saveCloutGraph(&dgraphset, dburl)

	return nil
}

func isTosca(metadata *ard.Map, etype string) bool {
	if *metadata != nil {
		if !reflect.ValueOf(metadata).IsNil() {

			entityType := (*metadata)["kind"]

			return (entityType == etype)
		}
	}

	return false
}
func isToscaVertex(vertex *clout.Vertex, etype string) bool {
	var data = vertex.Metadata["puccini-tosca"]
	if data != nil {
		var metadata = data.(ard.Map)
		return isTosca(&metadata, etype)
	}

	return false
}
func isToscaEdge(edge *clout.Edge, etype string) bool {
	var data = edge.Metadata["puccini-tosca"]
	if data != nil {
		var metadata = data.(ard.Map)
		return isTosca(&metadata, etype)
	}

	return false
}
func fillTosca(item *ard.Map, entity *ard.Map, type_ string, prefix string) error {
	//if prefix == nil {
	//	prefix = "";
	//}
	(*item)[prefix+"tosca:entity"] = type_
	(*item)[prefix+"tosca:name"] = (*entity)["name"]
	(*item)[prefix+"tosca:description"] = (*entity)["description"]
	mapx := ((*entity)["types"]).(ard.Map)
	bytes, error := json.Marshal(mapx)
	if error == nil {
		(*item)[prefix+"tosca:types"] = string(bytes)
	}
	mapx = ((*entity)["properties"]).(ard.Map)
	//	propmap := make(ard.Map)
	//	for key, valuemap := range mapx {
	//		propmap[key] = valuemap.(ard.Map)["value"]
	//	}
	//	bytes, error = json.Marshal(propmap)
	bytes, error = json.Marshal(mapx)
	if error == nil {
		(*item)[prefix+"tosca:properties"] = string(bytes)
		//(*item)[prefix+"tosca:properties"] = mapx
	}
	if (*entity)["attributes"] != nil {
		mapx = (*entity)["attributes"].(ard.Map)
		//	propmap = make(ard.Map)
		//	for key, valuemap := range mapx {
		//		propmap[key] = valuemap.(ard.Map)["value"]
		//	}
		//	bytes, error = json.Marshal(propmap)
		bytes, error = json.Marshal(mapx)

		if error == nil {
			(*item)[prefix+"tosca:attributes"] = string(bytes)
			//(*item)[prefix+"tosca:attributes"] = mapx
		}
	}

	return nil
}

func fillNodeTemplate(item *ard.Map, nodeTemplate *ard.Map) error {
	fillTosca(item, nodeTemplate, "nodeTemplate", "")

	itemCapabilities := make([]ard.Map, 0)
	var capabilities ard.Map = (*nodeTemplate)["capabilities"].(ard.Map)
	var cap ard.Map
	for _, capability := range capabilities {
		cap = capability.(ard.Map)
		capabilityItem := make(ard.Map)
		fillTosca(&capabilityItem, &cap, "capability", "")
		itemCapabilities = append(itemCapabilities, capabilityItem)
	}

	(*item)["capabilities"] = itemCapabilities

	return nil
}
func fillEdge(item *ard.Map, edge *clout.Edge) error {

	edgeItem := make(ard.Map)
	edgeItem["uid"] = "_:clout.vertex." + edge.Target.ID

	if isToscaEdge(edge, "relationship") {
		fillRelationship(&edgeItem, &edge.Properties)
	}

	var edgeItems []*ard.Map
	var edges = (*item)["clout:edge"]
	if edges == nil {
		edgeItems = make([]*ard.Map, 0)
	} else {
		edgeItems = (*item)["clout:edge"].([]*ard.Map)
	}

	(*item)["clout:edge"] = append(edgeItems, &edgeItem)

	return nil
}

func fillRelationship(item *ard.Map, relationship *ard.Map) error {
	// As facets
	fillTosca(item, relationship, "relationship", "clout:edge|")

	return nil
}

// declare cancel function for dgraph client connection
type CancelFunc func()

// Fetch a Dgraph Go client connected to a Dgraph db
func getDgraphClient(dburl string, isEnterprise bool) (*dgo.Dgraph, CancelFunc) {
	conn, err := grpc.Dial(dburl, grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	// Perform login call. If the Dgraph cluster does not have ACL and
	// enterprise features enabled, this call should be skipped.
	if isEnterprise {
		ctx := context.Background()
		for {
			// Keep retrying until we succeed or receive a non-retriable error.
			err = dg.Login(ctx, "groot", "password")
			if err == nil || !strings.Contains(err.Error(), "Please retry") {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			log.Fatalf("While trying to login %v", err.Error())
		}
	}

	return dg, func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error while closing connection:%v", err)
		}
	}
}

// clear Dgraph database
func dgraphDropAll(dburl string) {
	dg, cancel := getDgraphClient(dburl, false)
	defer cancel()
	op := api.Operation{DropAll: true}
	ctx := context.Background()
	if err := dg.Alter(ctx, &op); err != nil {
		log.Fatal(err)
	}
	// Output:
}

// Save the clout dgraph set in Dgraph db
func saveCloutGraph(dgraphSet *DgraphSet, dburl string) {
	//dburl = "localhost:9080"
	conn, err := grpc.Dial(dburl, grpc.WithInsecure())
	if err != nil {
		common.FailOnError(err)
	}
	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	mu := &api.Mutation{
		CommitNow: true,
	}
	cloutJSON, err := json.Marshal(*dgraphSet)
	if err != nil {
		common.FailOnError(err)
	}

	ctx := context.Background()
	mu.SetJson = cloutJSON
	assigned, err := dgraphClient.NewTxn().Mutate(ctx, mu)
	if err != nil {
		common.FailOnError(err)
	} else {
		fmt.Println("Assigned UUIDs : ")
		for key, value := range assigned.Uids {
			fmt.Println("name:", key, ",  value:", value)
		}
	}
}

func extractTopologyName(urlString string) string {

	ind := strings.LastIndex(urlString, "/")

	var topologyName string = urlString
	if ind == -1 {
		ind = strings.LastIndex(urlString, "\\")
	}
	if ind > -1 {
		topologyName = urlString[ind+1:]
	}
	ind = strings.LastIndex(topologyName, ".")
	if ind > -1 {
		topologyName = topologyName[:ind]
	}

	return topologyName
}
