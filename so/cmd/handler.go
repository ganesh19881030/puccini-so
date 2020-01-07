package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/gorilla/mux"
	"github.com/tliron/puccini/clout"

	//"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/url"

	//"google.golang.org/grpc"
	//"log"
	"io/ioutil"
	"net/http"
)

type Template struct {
	Uid            string                   `json:"uid" yaml:"uid"`
	Name           string                   `json:"name" yaml:"name"`
	Version        string                   `json:"version" yaml:"version"`
	GrammarVersion string                   `json:"grammarversion" yaml:"grammarversion"`
	Properties     interface{}              `json:"properties" yaml:"properties"`
	Vertexes       []map[string]interface{} `json:"vertexes" yaml:"vertexes"`
}

type Service struct {
	Uid          string                   `json:"uid" yaml:"uid"`
	Name         string                   `json:"name" yaml:"name"`
	Type         string                   `json:"type" yaml:"type"`
	TemplateName string                   `json:"templateName" yaml:"templateName"`
	Properties   interface{}              `json:"properties" yaml:"properties"`
	Vertexes     []map[string]interface{} `json:"vertexes" yaml:"vertexes"`
}

type Response struct {
	Result  string `json:"result" yaml:"result"`
	Message string `json:"message" yaml:"message"`
}

func HandleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/bonap/templates", getAllTemplates).Methods("GET")
	myRouter.HandleFunc("/bonap/templates/{name}", getTemplateByName).Methods("GET")
	myRouter.HandleFunc("/bonap/templates/{name}/{function}", executeFunction).Methods("POST")
	//myRouter.HandleFunc("/bonap/templates/{name}/createInstance/{service}", createInstance).Methods("POST")
	myRouter.HandleFunc("/bonap/templates/{name}/workflows", getWorkflows).Methods("GET")
	myRouter.HandleFunc("/bonap/templates/{name}/workflows/{wfname}", executeWorkflow).Methods("POST")
	//myRouter.HandleFunc("/bonap/templates/{name}/services", getServices).Methods("GET")
	//myRouter.HandleFunc("/bonap/templates/{name}/services/{service}", getServiceByName).Methods("GET")
	//myRouter.HandleFunc("/bonap/templates/{name}/services/{service}/workflow/{wfname}", executeWorkflow).Methods("POST")
	myRouter.HandleFunc("/bonap/templates/{name}/policies", getPolicies).Methods("GET")
	myRouter.HandleFunc("/bonap/templates/{name}/services/{service}/policy/{pname}", executePolicy).Methods("POST")
	myRouter.HandleFunc("/bonap/templates/{name}/services/{service}/policy/{pname}", stopPolicyExecution).Methods("DELETE")
	log.Info("Starting server at port 10000")
	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

/*func createConnection() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:9082", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return conn
}*/

func getAllTemplates(w http.ResponseWriter, r *http.Request) {

	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	const q = `
	{
		all(func: has(<clout:grammarversion>)){
			uid
			<clout:name>
			<clout:version>
			<clout:grammarversion>
			<clout:properties>
			<clout:vertex> {
				<tosca:name>
				<tosca:entity>
				<tosca:attributes>
			}
		}
	}`

	resp, err := txn.Query(context.Background(), q)
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}

	tmpls := make([]Template, 0)

	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {

		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			vertexList := cloutMap["clout:vertex"].([]interface{})
			nodeList := make([]map[string]interface{}, 0)
			for _, ver := range vertexList {
				vertex := ver.(map[string]interface{})
				node := make(map[string]interface{})
				//node["id"] = vertex["tosca:vertexId"]
				if vertex["tosca:name"] != nil {
					node["name"] = vertex["tosca:name"]
				}
				if vertex["tosca:entity"] != nil {
					node["entity"] = vertex["tosca:entity"]
				}
				node["attributes"] = getPropMap(vertex["tosca:attributes"])
				nodeList = append(nodeList, node)
			}
			tmpl := Template{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:version"].(string),
				cloutMap["clout:grammarversion"].(string), getPropMap(cloutMap["clout:properties"]),
				nodeList}

			tmpls = append(tmpls, tmpl)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(tmpls)
	w.Write(output)

}

func getTemplateByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)
	// Query the clout vertex by name
	const q = `query all($name: string) {
		all(func: eq(<clout:name>, $name)) {
			uid
			<clout:name>
			<clout:version>
			<clout:grammarversion>
			<clout:properties>   
			<clout:vertex>  {
				<tosca:name>
				<tosca:entity>
				<tosca:attributes>
			}
	    }
	}`
	resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})
	//if err != nil {
	//	log.Fatal(err)
	//}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}
	queryData := result["all"].([]interface{})
	if len(queryData) <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		writeResponse(Response{"Failure", "Template not found."}, w)
		return
	}
	cloutMap := queryData[0].(map[string]interface{})
	vertexList := cloutMap["clout:vertex"].([]interface{})
	nodeList := make([]map[string]interface{}, 0)
	for _, ver := range vertexList {
		vertex := ver.(map[string]interface{})
		node := make(map[string]interface{})
		node["id"] = vertex["tosca:vertexId"]
		if vertex["tosca:name"] != nil {
			node["name"] = vertex["tosca:name"]
		}
		if vertex["tosca:entity"] != nil {
			node["entity"] = vertex["tosca:entity"]
		}
		node["attributes"] = getPropMap(vertex["tosca:attributes"])
		/*if vertex["tosca:firstStep"] != nil {
			node["firstStep"] = vertex["tosca:firstStep"]
		}*/
		nodeList = append(nodeList, node)
	}
	tmpl := Template{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:version"].(string),
		cloutMap["clout:grammarversion"].(string), getPropMap(cloutMap["clout:properties"]),
		nodeList}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(tmpl)
	w.Write(output)

}

func executeFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	fn := vars["function"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//common.FailOnError(err)

	// Try loading JavaScript from Clout
	sourceCode, err := js.GetScriptSourceCode(fn, clout_)

	if err != nil {
		// Try loading JavaScript from path or URL
		url_, err := url.NewValidURL(fn, nil)
		if err != nil {
			writeResponse(Response{"Failure", fn + " URL not found."}, w)
			return
		}
		sourceCode, err = url.Read(url_)
		if err != nil {
			writeResponse(Response{"Failure", err.Error()}, w)
			return
		}

		err = js.SetScriptSourceCode(fn, js.Cleanup(sourceCode), clout_)
		if err != nil {
			writeResponse(Response{"Failure", err.Error()}, w)
			return
		}

	}

	err = Exec(fn, sourceCode, clout_)
	if err != nil {
		writeResponse(Response{"Failure", err.Error()}, w)
	} else {
		writeResponse(Response{"Success", fn + " executed successfully"}, w)
	}

}

// Handles request "/bonap/templates/{name}/workflows"
func getWorkflows(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	//wfName := vars["wfname"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//Insert workflow steps inside clout
	updateCloutWithWorkflows(clout_)

	// Write into a file
	file, _ := json.MarshalIndent(clout_, "", "  ")
	_ = ioutil.WriteFile("fw1_csar_dgraph_wf.json", file, 0644)

	// Process Workflow by name
	workflows := CreateWorkflows(clout_)
	if workflows == nil {
		writeResponse(Response{"Failure", "No workflows found"}, w)
		return
	}
	//if err != nil {
	//	writeResponse(Response{"Failure", err.Error()}, w)
	//} else {
	//writeResponse(Response{"Success", "Workflow [" + wfName + "] executed successfully"}, w)
	//}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(workflows)
	if err != nil {
		fmt.Println(err.Error())
		writeResponse(Response{"Failure", "Error creating workflows"}, w)
		return
	}
	w.Write(output)

}

// Handles request "/bonap/templates/{name}/workflows/{wfname}"
func executeWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	//service := vars["service"]
	wfName := vars["wfname"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//Insert workflow steps inside clout
	updateCloutWithWorkflows(clout_)

	// Process Workflow by name
	workflows := CreateWorkflows(clout_)
	if workflows == nil {
		writeResponse(Response{"Failure", "No workflows found"}, w)
		return
	}

	workflow := workflows[wfName]
	if workflow == nil {
		writeResponse(Response{"Failure", "Workflow [" + wfName + "] found"}, w)
		return
	}

	wferr := ExecuteWorkflow(workflow, name)
	if wferr != nil {
		//fmt.Println(err.Error{})
		writeResponse(Response{"Failure", "Error creating workflow [" + wfName + "]"}, w)
		return
	}
	writeResponse(Response{"Success", "Workflow [" + wfName + "] executed successfully"}, w)
	//}

}

// Handles request "/bonap/templates/{name}/policies"
func getPolicies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	policies := createPolicies(clout_)

	if policies == nil {
		writeResponse(Response{"Failure", "No Policy found"}, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(policies)
	if err != nil {
		fmt.Println(err.Error())
		writeResponse(Response{"Failure", "Error getting policies"}, w)
		return
	}
	w.Write(output)

}

func executePolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	//service := vars["service"]
	pname := vars["pname"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	policies := createPolicies(clout_)

	policiesDef := policies.PolicyDefinitions
	policy := policiesDef[pname]
	if policy == nil {
		writeResponse(Response{"Failure", "No Policy found"}, w)
		return
	}

	wferr := ExecutePolicy(policy)
	if wferr != nil {
		writeResponse(Response{"Failure", "Error creating Policy [" + pname + "]"}, w)
		return
	}
	writeResponse(Response{"Success", "Policy [" + pname + "] executed successfully"}, w)
}

func stopPolicyExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	//service := vars["service"]
	pname := vars["pname"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	policies := createPolicies(clout_)

	policiesDef := policies.PolicyDefinitions
	policy := policiesDef[pname]
	if policy == nil {
		writeResponse(Response{"Failure", "No Policy found"}, w)
		return
	}

	wferr := DeletePolicy(policy)
	if wferr != nil {
		writeResponse(Response{"Failure", "Error deleting Policy [" + pname + "]"}, w)
		return
	}
	writeResponse(Response{"Success", "Policy [" + pname + "] deleted successfully"}, w)
}

/*func executeWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	wfName := vars["wfname"]

	//Read Clout from Dgraph
	clout_, _, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	// Process Workflow by name
	// Process Workflow by name
	workflows := CreateWorkflows(clout_)
	if workflows == nil {
		writeResponse(Response{"Failure", "No workflows found"}, w)
		return
	}

	workflow := workflows[wfName]
	if workflow == nil {
		writeResponse(Response{"Failure", "Workflow [" + wfName + "] found"}, w)
		return
	}

	wferr := ExecuteWorkflow(workflow)
	if wferr != nil {
		//fmt.Println(err.Error{})
		writeResponse(Response{"Failure", "Error creating workflow [" + wfName + "]"}, w)
		return
	}
	writeResponse(Response{"Success", "Workflow [" + wfName + "] executed successfully"}, w)
	//}

}*/

func createInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	service := vars["service"]
	//fn := vars["function"]

	//Read Clout from Dgraph
	clout_, uid, err := ReadCloutFromDgraph(name)
	if clout_ == nil || err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = CreateInstance(clout_, name, service, uid)
	if err != nil {
		writeResponse(Response{"Failure", err.Error()}, w)
	} else {
		writeResponse(Response{"Success", "Instance created successfully"}, w)
	}

}

func getServices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	const q = `query all($name: string) {
		all(func: eq(<clout:templateName>, $name)) {
			uid
			<clout:name>
			<clout:properties>
    		<clout:type>
    		<clout:templateName>
			<clout:vertex> {
				<tosca:name>
				<tosca:entity>
				<tosca:attributes>
			}
	    }
	}`
	resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})
	//if err != nil {
	//	log.Fatal(err)
	//}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}

	services := make([]Service, 0)

	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {

		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			vertexList := cloutMap["clout:vertex"].([]interface{})
			nodeList := make([]map[string]interface{}, 0)
			for _, ver := range vertexList {
				vertex := ver.(map[string]interface{})
				node := make(map[string]interface{})
				//node["id"] = vertex["tosca:vertexId"]
				if vertex["tosca:name"] != nil {
					node["name"] = vertex["tosca:name"]
				}
				if vertex["tosca:entity"] != nil {
					node["entity"] = vertex["tosca:entity"]
				}
				if vertex["tosca:attributes"] != nil {
					node["attributes"] = getAttributes(vertex["tosca:attributes"])
				}
				nodeList = append(nodeList, node)
			}
			serv := Service{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:type"].(string),
				cloutMap["clout:templateName"].(string), getPropMap(cloutMap["clout:properties"]),
				nodeList}

			services = append(services, serv)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(services)
	w.Write(output)

}

func getServiceByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateName := vars["name"]
	serviceName := vars["service"]

	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	const q = `query all($name: string, $sname: string, $stype: string) {
		all(func: eq(<clout:templateName>, $name)) @filter (eq(<clout:name>, $sname) 
			AND eq(<clout:type>, $stype)){
			uid
			<clout:name>
			<clout:properties>
    		<clout:type>
    		<clout:templateName>
			<clout:vertex> {
				<tosca:name>
				<tosca:entity>
				<tosca:attributes>
			}
	    }
	}`
	resp, err := txn.QueryWithVars(context.Background(), q,
		map[string]string{"$name": templateName, "$sname": serviceName, "$stype": "service"})
	//if err != nil {
	//	log.Fatal(err)
	//}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}

	//services := make([]Service, 0)
	var service Service

	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {

		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			if cloutMap["clout:name"].(string) == serviceName {
				vertexList := cloutMap["clout:vertex"].([]interface{})
				nodeList := make([]map[string]interface{}, 0)
				for _, ver := range vertexList {
					vertex := ver.(map[string]interface{})
					node := make(map[string]interface{})
					node["id"] = vertex["tosca:vertexId"]
					if vertex["tosca:name"] != nil {
						node["name"] = vertex["tosca:name"]
					}
					if vertex["tosca:entity"] != nil {
						node["entity"] = vertex["tosca:entity"]
					}
					if vertex["tosca:attributes"] != nil {
						node["attributes"] = getAttributes(vertex["tosca:attributes"])
					}
					nodeList = append(nodeList, node)
				}
				service = Service{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:type"].(string),
					cloutMap["clout:templateName"].(string), getPropMap(cloutMap["clout:properties"]),
					nodeList}
				break
			}

			//services = append(services, serv)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(service)
	w.Write(output)

}

func getAttributes(attrs interface{}) map[string](map[string]interface{}) {
	attrMap := getPropMap(attrs)
	attributes := make(map[string](map[string]interface{}))
	for k, attr := range attrMap {
		attrib := attr.(map[string]interface{})
		a := make(map[string]interface{})
		//attributes[k] = make(map[string]interface{})
		//fmt.Println(attrib)
		a["value"] = attrib["value"]
		attributes[k] = a
	}
	return attributes

}

func updateCloutWithWorkflows(clout_ *clout.Clout) {
	// create workflow steps
	workFlows := createWorkFlows(clout_)

	// store workflow steps into clout structure
	storeWorkflowsIntoClout(workFlows.WorkflowDefinitions, clout_)

}

func writeResponse(res Response, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	output, err := json.Marshal(res)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(output)
}
