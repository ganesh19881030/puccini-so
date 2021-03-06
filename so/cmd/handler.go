package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/gorilla/mux"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/so/db"
	"github.com/tliron/puccini/tosca/database"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
	"github.com/tliron/puccini/tosca/normal"
	"github.com/tliron/puccini/url"
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
	myRouter.HandleFunc("/bonap/templates/createInstance", createInstance).Methods("POST")
	//myRouter.HandleFunc("/bonap/templates/{name}/workflows", getWorkflows).Methods("GET")
	myRouter.HandleFunc("/bonap/templates/workflows", getWorkflows).Methods("POST")
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

	dgt, err := connectToDgraph(w)
	if err != nil {
		return
	}
	defer dgt.Close()

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

	//resp, err := txn.Query(context.Background(), q)
	resp, err := dgt.ExecQuery(q)
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return
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
				node["attributes"], err = getPropMap(vertex["tosca:attributes"])
				if err != nil {
					log.Errorf(err.Error())
					writeResponse(Response{"Failure", err.Error()}, w)
					return
				}

				nodeList = append(nodeList, node)
			}
			propMap, err := getPropMap(cloutMap["clout:properties"])
			if err != nil {
				log.Errorf(err.Error())
				writeResponse(Response{"Failure", err.Error()}, w)
				return
			}

			tmpl := Template{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:version"].(string),
				cloutMap["clout:grammarversion"].(string), propMap,
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

	dgt, err := connectToDgraph(w)
	if err != nil {
		return
	}
	defer dgt.Close()
	/*
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
				}
		    }
		}`
		resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": name})
	*/
	const paramq = `
	{
		all(func: has(<clout:grammarversion>)) @filter(eq(<clout:name>,"%s")){
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
	query := fmt.Sprintf(paramq, name)
	resp, err := dgt.ExecQuery(query)
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
		node["attributes"], err = getPropMap(vertex["tosca:attributes"])
		if err != nil {
			log.Errorf(err.Error())
			writeResponse(Response{"Failure", err.Error()}, w)
			return
		}
		/*if vertex["tosca:firstStep"] != nil {
			node["firstStep"] = vertex["tosca:firstStep"]
		}*/
		nodeList = append(nodeList, node)
	}
	propMap, err := getPropMap(cloutMap["clout:properties"])
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return
	}
	tmpl := Template{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:version"].(string),
		cloutMap["clout:grammarversion"].(string), propMap,
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
/*func getWorkflows(w http.ResponseWriter, r *http.Request) {
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

}*/

// Handles request "/bonap/templates/workflows"
func getWorkflows(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		eMsg := "Empty request body is not allowed!"
		log.Errorf(eMsg)
		writeResponse(Response{"Failure", eMsg}, w)
		return
	}
	var params ard.Map
	//json.Unmarshal(r.Body, &m)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		eMsg := err.Error()
		log.Errorf(eMsg)
		writeResponse(Response{"Failure", eMsg}, w)
		return
	}

	cdresult := createCloutFromDgraph(w, req, params, false)

	//Insert workflow steps inside clout
	updateCloutWithWorkflows(cdresult.Clout)

	// Write into a file
	file, _ := json.MarshalIndent(cdresult.Clout, "", "  ")
	_ = ioutil.WriteFile("fw1_csar_dgraph_wf.json", file, 0644)

	// Process Workflow by name
	workflows := CreateWorkflows(cdresult.Clout)
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

	wferr := ExecuteWorkflow(workflow)
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
		//w.WriteHeader(http.StatusNotFound)
		writeResponse(Response{"Failure", err.Error()}, w)
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

/*
   createInstance handles POST request in the following form:

   /bonap/templates/createInstance

   It expects the POST request body to contain, for example, the following json body:

{
	"name" : "zip:/Users/rajee/git/workdir/firewall.csar!/firewall/firewall_service.yaml",
	"output": "../../workdir/fw-dgraph-clout.yaml",
	"inputs": {
		"selected_flavour":"simple",
		"region_name":"DFW",
		"lower_threshold":"10",
		"upper_threshold":"80",
		"packet_rate":"20",
		"cidr":"192.168.1.0",
		"network_name":"public",
		"num_streams":"5",
		"auth_url":"http://localhost/",
		"password":"password",
		"project_id":"101012",
		"url":"http://localhost",
		"username":"cci"
	},
	"quirks": ["data_types.string.permissive"],
	"inputsUrl":"",
	"generate-workflow":false,
	"execute-workflow":false,
	"service":"firewall_service"
}
*/
func createInstance(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		eMsg := "Empty request body is not allowed!"
		log.Errorf(eMsg)
		writeResponse(Response{"Failure", eMsg}, w)
		return
	}
	var params ard.Map
	//json.Unmarshal(r.Body, &m)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		eMsg := err.Error()
		log.Errorf(eMsg)
		writeResponse(Response{"Failure", eMsg}, w)
		return
	}

	genWf, err := checkRequestParameterBoolean("generate-workflow", &params)
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return
	}

	execWf, err := checkRequestParameterBoolean("execute-workflow", &params)
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return
	}

	cdresult := createCloutFromDgraph(w, req, params, true)

	//clout_, uid, err := ReadCloutFromDgraph(name)
	//if clout == nil || err != nil {
	if cdresult == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//Insert workflow steps inside clout
	if genWf {
		updateCloutWithWorkflows(cdresult.Clout)

		if execWf {
			// Process Workflow by name
			workflows := CreateWorkflows(cdresult.Clout)
			if workflows == nil {
				writeResponse(Response{"Failure", "No workflows found"}, w)
				return
			}

			wfName := "deploy"
			workflow := workflows[wfName]
			if workflow == nil {
				writeResponse(Response{"Failure", "Workflow [" + wfName + "] found"}, w)
				return
			}

			wferr := ExecuteWorkflow(workflow)
			if wferr != nil {
				writeResponse(Response{"Failure", "Error creating workflow [" + wfName + "]"}, w)
				return
			}
		}
	}

	//err = CreateInstance(clout_, name, service, uid)
	err = saveCloutInDgraph(cdresult)

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
					node["attributes"], err = getAttributes(vertex["tosca:attributes"])
					if err != nil {
						writeResponse(Response{"Failure", err.Error()}, w)
						return
					}
				}
				nodeList = append(nodeList, node)
			}
			propMap, err := getPropMap(cloutMap["clout:properties"])
			if err != nil {
				writeResponse(Response{"Failure", err.Error()}, w)
				return
			}

			serv := Service{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:type"].(string),
				cloutMap["clout:templateName"].(string), propMap,
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
						node["attributes"], err = getAttributes(vertex["tosca:attributes"])
						if err != nil {
							writeResponse(Response{"Failure", err.Error()}, w)
							return
						}
					}
					nodeList = append(nodeList, node)
				}
				propMap, err := getPropMap(cloutMap["clout:properties"])
				if err != nil {
					writeResponse(Response{"Failure", err.Error()}, w)
					return
				}

				service = Service{cloutMap["uid"].(string), cloutMap["clout:name"].(string), cloutMap["clout:type"].(string),
					cloutMap["clout:templateName"].(string), propMap,
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

func getAttributes(attrs interface{}) (map[string](map[string]interface{}), error) {
	attrMap, err := getPropMap(attrs)
	if err != nil {
		return nil, err
	}
	attributes := make(map[string](map[string]interface{}))
	for k, attr := range attrMap {
		attrib := attr.(map[string]interface{})
		a := make(map[string]interface{})
		//attributes[k] = make(map[string]interface{})
		//fmt.Println(attrib)
		a["value"] = attrib["value"]
		attributes[k] = a
	}
	return attributes, nil

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

func connectToDgraph(w http.ResponseWriter) (*dgraph.DgraphTemplate, error) {
	dgt, err := fetchDbTemplate()
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
	}
	return dgt, err
}

type CloutDgraphResult struct {
	Clout           *clout.Clout
	Dbc             *db.DgContext
	ServiceTemplate *normal.ServiceTemplate
	Sturl           url.URL
}

func createCloutFromDgraph(w http.ResponseWriter, req *http.Request, params ard.Map, isCreate bool) *CloutDgraphResult {

	cdresult := new(CloutDgraphResult)

	name := params["name"].(string)
	//service := params["service"].(string)
	output := params["output"].(string)
	inputsUrl := params["inputsUrl"].(string)
	var inputValues ard.Map
	if inputsUrl != "" {
		inputValues = ParseInputsFromUrl(inputsUrl)
	}
	inputs := params["inputs"].(ard.Map)
	if len(inputs) > 0 {
		inputValues = make(ard.Map)
		for key, val := range inputs {
			value, err := format.Decode(val.(string), "yaml")
			if err != nil {
				log.Errorf(err.Error())
				writeResponse(Response{"Failure", err.Error()}, w)
				return nil
			}
			inputValues[key] = value
		}
	}
	var quirks []string
	if quirkparams, ok := params["quirks"].([]interface{}); ok {
		for _, quirkparam := range quirkparams {
			if quirkstr, ok := quirkparam.(string); ok {
				quirks = append(quirks, quirkstr)
			}
		}
	}

	//fn := vars["function"]

	//Read Clout from Dgraph

	if isCreate && CloutInstanceExists(name) {
		emsg := fmt.Sprintf("Clout instance with name %s already exists!", name)
		log.Errorf(emsg)
		writeResponse(Response{"Failure", emsg}, w)
		return nil
	}

	urlst, err := url.NewValidURL(name, nil)
	if err != nil {
		log.Errorf(err.Error())
		writeResponse(Response{"Failure", err.Error()}, w)
		return nil
	}

	dbc := new(db.DgContext)
	st, ok := dbc.ReadServiceTemplateFromDgraph(urlst, inputValues, quirks)
	var clout *clout.Clout
	if !ok {
		return nil
	} else {
		clout, err = dbc.Compile(st, urlst, resolve, coerce, output, false)
		if err != nil {
			log.Errorf(err.Error())
			writeResponse(Response{"Failure", err.Error()}, w)
			return nil
		}
	}

	cdresult.Clout = clout
	cdresult.Dbc = dbc
	cdresult.Sturl = urlst
	cdresult.ServiceTemplate = st

	return cdresult
}

func saveCloutInDgraph(cdresult *CloutDgraphResult) error {
	internalImport := common.InternalImport
	urlString := strings.Replace(cdresult.Sturl.String(), "\\", "/", -1)
	return database.Persist(cdresult.Clout, cdresult.ServiceTemplate, urlString, cdresult.Dbc.Pcontext.GrammerVersions, internalImport)
}

func checkRequestParameterBoolean(paramkey string, params *map[string]interface{}) (bool, error) {
	var retValue bool
	var err error
	val, ok := (*params)[paramkey]
	if ok {
		retValue, ok = val.(bool)
		if !ok {
			err = errors.New(paramkey + " must be a boolean value")
		}
	} else {
		err = errors.New(paramkey + " parameter is missing")
	}
	return retValue, err
}
