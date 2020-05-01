package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	flow "github.com/tliron/puccini/wf/components"
	"golang.org/x/crypto/ssh"

	//"os/exec"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var instanceName string
var templateName string
var uid string

//var inputs ard.Map
var baseClout *clout.Clout

type Node struct {
	Uid        string `json:"uid" yaml:"uid"`
	Attributes string `json:"tosca:attributes" yaml:"tosca:attributes"`
}

func CreateWorkflows(clout *clout.Clout) map[string]*flow.Workflow {

	baseClout = clout

	var vertexes = extractWorkflowsFromClout(clout)

	if len(vertexes) <= 0 {
		fmt.Println("No workflows defined")
		return nil
		//return flow.NewWfError("No workflows defined")
	}

	workflows := make(map[string]*flow.Workflow, 0)
	//var procNames []string

	for _, vertex := range vertexes {
		wf := vertex.Properties
		wfname := wf["name"].(string)
		workflow := flow.NewWorkflow(wfname, 0)

		// Iterate steps
		procNames := make([]string, 0)
		var success string
		var failure string
		var successes []string
		var failures []string
		count := 0
		for _, edge := range vertex.EdgesOut {
			if isToscaEdge(edge, "workflowStep") {
				//var stepID = edge.TargetID
				//workflowStep := getVertex(clout, stepID, "workflowStep")
				workflowStep := edge.Target
				name := workflowStep.Properties["name"].(string)
				successes = make([]string, 0)
				failures = make([]string, 0)
				for _, ed := range workflowStep.EdgesOut {
					if isToscaEdge(ed, "onSuccess") {
						//var ID = ed.TargetID
						//vert := getVertex(clout, ID, "workflowStep")
						vert := ed.Target
						success = vert.Properties["name"].(string)
						successes = append(successes, success)
						procNames = append(procNames, success)
					} else if isToscaEdge(ed, "onFailure") {
						//var ID = ed.TargetID
						//vert := getVertex(clout, ID, "workflowStep")
						vert := ed.Target
						failure = vert.Properties["name"].(string)
						failures = append(failures, failure)
						procNames = append(procNames, failure)
					}
				}
				//firstStep := workflowStep.Properties["firstStep"].(bool)
				workflowProcess := flow.NewProcess(workflow, name, successes, failures)
				createWorkflowTask(workflowStep, workflowProcess, clout)
				count++
			}

		}
		workflow.SetCount(count)
		workflows[wfname] = workflow
		setStartProcess(workflow, procNames)
	}
	//err := workflows[wfn].Run()
	return workflows
}

/*func ExecuteWorkflow(wfn string, workflows map[string]*flow.Workflow) *flow.WfError {
	err := workflows[wfn].Run()
	return err
}*/

/*func ExecuteWorkflow(workflow *flow.Workflow, tname string, sname string) *flow.WfError {
	instanceName = sname
	templateName = tname
	//uid = getServiceUID(templateName, instanceName)
	uid, inputs = getServiceInputs(templateName, instanceName)
	err := workflow.Run()
	return err
}*/

//func ExecuteWorkflow(workflow *flow.Workflow, tname string) *flow.WfError {
func ExecuteWorkflow(workflow *flow.Workflow) *flow.WfError {
	var wg sync.WaitGroup
	//instanceName = sname
	//templateName = tname
	//uid = getServiceUID(templateName, instanceName)
	//uid, inputs = getServiceInputs(templateName)
	err := workflow.Run(&wg)
	fmt.Println("Main: Waiting for threads to finish")
	currentTime := time.Now()
	fmt.Println("Start time: ", currentTime.String())
	wg.Wait()
	fmt.Println("Main: Completed")
	currentTime = time.Now()
	fmt.Println("End time: ", currentTime.String())
	file, _ := json.MarshalIndent(baseClout, "", "  ")
	_ = ioutil.WriteFile("fw1_csar_dgraph_final.json", file, 0644)

	return err
}

func setStartProcess(wf *flow.Workflow, procNames []string) {
	procs := wf.GetProcs()
	for key, proc := range procs {
		if !isFound(key, procNames) {
			proc.SetStart(true)
			//break
		}
	}
}

/*func getUID(tname string, sname string) string {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	const q = `query all($name: string) {
		all(func: eq(<clout:templateName>, $name)) @filter (eq(<clout:name>, $sname)
			AND eq(<clout:type>, $stype)){
			uid
	    }
	}`
	resp, err := txn.QueryWithVars(context.Background(), q,
		map[string]string{"$name": templateName, "$sname": serviceName, "$stype": "service"})

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}
	uid := ""

	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {
		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			uid = cloutMap["uid"].(string)
		}
	}
	return uid

}*/

func isFound(name string, procNames []string) bool {
	for _, procName := range procNames {
		if procName == name {
			return true
		}
	}

	return false
}

func createWorkflowTask(wfStep *clout.Vertex, workflowProcess *flow.Process,
	clout1 *clout.Clout) {
	nodeTemplates := make([]*clout.Vertex, 0)
	groups := make([]*clout.Vertex, 0)
	activities := make(map[int]*clout.Vertex)

	for _, edge := range wfStep.EdgesOut {
		if isToscaEdge(edge, "nodeTemplateTarget") {
			//var ID = edge.TargetID
			//nodeTemplate := getVertex(clout1, ID, "nodeTemplate")
			nodeTemplate := edge.Target
			nodeTemplates = append(nodeTemplates, nodeTemplate)
		} else if isToscaEdge(edge, "groupTarget") {
			//var ID = edge.TargetID
			//group := getVertex(clout1, ID, "group")
			group := edge.Target
			groups = append(groups, group)
		} else if isToscaEdge(edge, "workflowActivity") {
			//var ID = edge.TargetID
			//activity := getVertex(clout1, ID, "workflowActivity")
			activity := edge.Target
			//sequence := edge.Properties["sequence"].(float64)
			sequence := edge.Properties["sequence"].(int)
			activities[int(sequence)] = activity
		}

	}

	keys := getSortedSequence(activities)

	for seq := range keys {
		act := activities[seq]
		activity := act.Properties
		if activity["setNodeState"] != nil {
			state := activity["setNodeState"].(string)
			//params := createParams(nodeTemplates, groups, activity)
			//flow.NewTask(workflowProcess, "setNodeState to "+state, seq, params, SetNodeState)
			/*flow.NewTask(workflowProcess, "setNodeState to "+state, seq, params,
				func() *flow.WfError {
					return SetNodeState(nodeTemplates, groups, activity)
				},
			)*/
			flow.NewTask(workflowProcess, "setNodeState to "+state, seq,
				func() *flow.WfError {
					return SetNodeState(nodeTemplates, groups, activity)
				},
			)
		} else if activity["callOperation"] != nil {
			oper := activity["callOperation"].(map[string]interface{})
			inter := oper["interface"].(string)
			operation := oper["operation"].(string)
			//params := createParams(nodeTemplates, groups, activity)
			/*flow.NewTask(workflowProcess, "interface: "+inter+"; operation: "+operation, seq, params,
				func() *flow.WfError {
					return CallOperation(nodeTemplates, groups, activity)
				},
			)*/
			flow.NewTask(workflowProcess, "interface: "+inter+"; operation: "+operation, seq,
				func() *flow.WfError {
					return CallOperation(clout1, nodeTemplates, groups, activity)
				},
			)
		}
	}

}

func createParams(nodeTemplates []*clout.Vertex, groups []*clout.Vertex, activity interface{}) map[string]interface{} {
	params := make(map[string]interface{})
	params["nodeTemplates"] = nodeTemplates
	params["groups"] = groups
	params["activity"] = activity

	return params
}

func getSortedSequence(m map[int]*clout.Vertex) []int {
	var keys []int
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func extractWorkflowsFromClout(clout1 *clout.Clout) []*clout.Vertex {
	vertexes := make([]*clout.Vertex, 0)
	for _, vertex := range clout1.Vertexes {
		if isToscaVertex(vertex, "workflow") {
			vertexes = append(vertexes, vertex)
		}
	}
	return vertexes
}

func getVertex(clout *clout.Clout, ID string, kind string) *clout.Vertex {
	for _, vertex := range clout.Vertexes {
		if isToscaVertex(vertex, kind) && vertex.ID == ID {
			return vertex
		}
	}
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

func SetNodeState(nodeTemplates []*clout.Vertex, groups []*clout.Vertex, activity interface{}) *flow.WfError {
	act := activity.(map[string]interface{})
	val := act["setNodeState"].(string)

	for _, nodeTemplate := range nodeTemplates {
		nodeName := nodeTemplate.Properties["name"].(string)
		//fmt.Println(nodeTemplate.Properties["name"].(string) + " set Node state ===> " + val)
		updateSingleNodeAttr(nodeName, "state", val)
	}

	/*for _, group := range groups {
		grpAttr := group.Properties["attributes"].(map[string]interface{})
		state := grpAttr["state"].(map[string]interface{})
		state["value"] = val
		fmt.Println(group.Properties["name"].(string) + " set Node state ===> " + val)
	}*/
	//fmt.Println(val)
	return nil
}

func updateSingleNodeAttr(nodeName string, key string, val string) error {
	result := getServiceNode(uid, nodeName)
	if result == nil || len(result) <= 0 {
		//fmt.Println("Node " + nodeName + " has no attribute called " + key)
		return nil
	}
	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {
		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			vertexList := cloutMap["clout:vertex"].([]interface{})
			vertex := vertexList[0].(map[string]interface{})
			nodeUid := vertex["uid"].(string)
			attrs, err := getPropMap(vertex["tosca:attributes"])
			if err != nil {
				return err
			}
			atr := attrs[key].(map[string]interface{})
			atr["value"] = val
			bytes, _ := json.Marshal(attrs)
			node := Node{Uid: nodeUid, Attributes: string(bytes)}
			updateNodeAttributes(node)
			//fmt.Println(node)
			//fmt.Println(atr)
		}
	}

	return nil
}

func updateNodeAttrs(nodeName string, resultMap map[string]interface{}, outputs map[string]interface{}) error {
	result := getServiceNode(uid, nodeName)
	if result == nil || len(result) <= 0 {
		//fmt.Println("Node " + nodeName + " has no attribute called " + key)
		return nil
	}
	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {
		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			vertexList := cloutMap["clout:vertex"].([]interface{})
			vertex := vertexList[0].(map[string]interface{})
			nodeUid := vertex["uid"].(string)
			attrs, err := getPropMap(vertex["tosca:attributes"])
			if err != nil {
				return err
			}
			for k, v := range outputs {
				data := v.(map[string]interface{})
				attrName := data["attributeName"].(string)
				atr := attrs[attrName].(map[string]interface{})
				atr["value"] = resultMap[k].(string)
			}
			bytes, _ := json.Marshal(attrs)
			node := Node{Uid: nodeUid, Attributes: string(bytes)}
			updateNodeAttributes(node)
		}
	}
	return nil
}

func CallOperation(clout1 *clout.Clout, nodeTemplates []*clout.Vertex, groups []*clout.Vertex,
	activity interface{}) *flow.WfError {
	act := activity.(map[string]interface{})
	callOp := act["callOperation"].(map[string]interface{})
	inter := callOp["interface"].(string)
	operName := callOp["operation"].(string)

	for _, tmpl := range nodeTemplates {
		nodeName := tmpl.Properties["name"].(string)
		nodeTemplate := getVertexByName(nodeName, "nodeTemplate")
		nodeInterface := nodeTemplate.Properties["interfaces"].(map[string]interface{})
		interf := nodeInterface[inter].(map[string]interface{})
		commonInputString := ""
		/*if strings.Contains(nodeName, "flavors__network_service.yaml.subnet") {
			fmt.Println("********************")
		}*/
		if interf["inputs"] != nil && len(interf["inputs"].(map[string]interface{})) > 0 {
			commonInputs := interf["inputs"].(map[string]interface{})
			commonInputString = getInputStrings(clout1, nodeTemplate, commonInputs)
		}
		opers := interf["operations"].(map[string]interface{})
		oper := opers[operName].(map[string]interface{})
		//name := nodeTemplate.Properties["name"].(string)
		//fmt.Println(name + " ===> " + oper["description"].(string))
		script := oper["implementation"].(string)
		//fmt.Println("                 " + script)
		if script != "" {
			inps := oper["inputs"].(map[string]interface{})
			inputString := commonInputString + " " + getInputStrings(clout1, nodeTemplate, inps)

			cmds := make([]string, 0)

			//cmds = append(cmds, "sudo -s source /opt/app/bonap/"+script+" "+inputString)
			cmds = append(cmds, "sudo /opt/app/bonap/"+script+" "+inputString)
			fmt.Println(cmds)
			//}
			resultMap := execRemoteCommand(cmds)
			if oper["outputs"] != nil {
				outputs := oper["outputs"].(map[string]interface{})
				updateNodeAttributeInClout(resultMap, outputs)
			}
		}

		//updateNodeData(nodeName, "public_address", ip)
	}

	for _, group := range groups {
		grpInterface := group.Properties["interfaces"].(map[string]interface{})
		inter := grpInterface[inter].(map[string]interface{})
		opers := inter["operations"].(map[string]interface{})
		oper := opers[operName].(map[string]interface{})
		fmt.Println(group.Properties["name"].(string) + " ===> " + oper["description"].(string))

	}

	return nil
}

func extractServerName(name string) string {
	names := strings.Split(name, ".")
	length := len(names)
	return names[length-1]
}

func execRemoteCommand(cmds []string) map[string]interface{} {
	//key, err := ssh.ParsePrivateKey([]byte(privateKey))
	user := common.SoConfig.Remote.RemoteUser
	addr := common.SoConfig.Remote.RemoteHost
	port := common.SoConfig.Remote.RemotePort
	pkfile := common.SoConfig.Remote.RemotePubKey
	pkBytes, err := ioutil.ReadFile(pkfile)
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
	}
	key, err := ssh.ParsePrivateKey(pkBytes)
	if err != nil {
		fmt.Println(err)
	}
	// Authentication
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		}),
		//alternatively, you could use a password
		/*
		   Auth: []ssh.AuthMethod{
		       ssh.Password("PASSWORD"),
		   },
		*/
	}
	// Connect
	client, err := ssh.Dial("tcp", net.JoinHostPort(addr, strconv.Itoa(port)), config)
	if err != nil {
		fmt.Println(err)
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		fmt.Println(err)
	}
	defer session.Close()
	var b bytes.Buffer   // import "bytes"
	var b1 bytes.Buffer  // import "bytes"
	session.Stderr = &b  // get error
	session.Stdout = &b1 // get output
	// you can also pass what gets input to the stdin, allowing you to pipe
	// content from client to server
	//      session.Stdin = bytes.NewBufferString("My input")

	// Finally, run the command
	cmd := strings.Join(cmds, "; ")
	if strings.Contains(cmd, "firewall/artifacts/ves/start.sh") {
		cmd = strings.Replace(cmd, "1.2.3.4", "3.135.237.9", 1)
		cmd = strings.Replace(cmd, "3904", "8081", 1)
		cmd = cmd + " \"report_file=firewall/artifacts/ves/cci_ves_reporter.sh\" "
		fmt.Println(cmd)
	}
	err = session.Run(cmd)

	if b.Len() > 0 {
		fmt.Println(b.String())
	}
	output := b1.String()
	fmt.Println(output)
	if output == "" {
		return nil
	}
	outputMap := make(map[string]interface{})
	err = json.Unmarshal(b1.Bytes(), &outputMap)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	return outputMap
}

func getInputStrings(clout1 *clout.Clout, nodeTemplate *clout.Vertex, inputs map[string]interface{}) string {
	inps := ""

	for key, v := range inputs {
		val := ""
		input := v.(map[string]interface{})
		data := extractInputData(clout1, nodeTemplate, input)

		if data != nil {
			if reflect.TypeOf(data).Kind() == reflect.Map {
				data1 := data.(map[string]interface{})
				val = fmt.Sprintf("%v", data1["originalString"])
			} else {
				val = fmt.Sprintf("%v", data)
			}
		}
		//if val != "" {
		inps = inps + " \"" + key + "=" + val + "\""
		//}
	}

	return inps

}

func extractInputData(clout1 *clout.Clout, nodeTemplate *clout.Vertex, input map[string]interface{}) interface{} {
	var node *clout.Vertex
	val, node := getValue(clout1, nodeTemplate, input)
	var name interface{}

	//node = nodeTemplate
	for {
		if val["value"] != nil {
			name = val["value"]
			break
		} else if val["functionCall"] != nil {
			var newval map[string]interface{}
			newval, node = getValue(clout1, node, val)
			val = newval
		} else if val["sourcePath"] != nil {
			name = val["sourcePath"]
			break
		} else if val == nil || val["value"] == nil {
			return nil
		}
	}

	return name
}

func getValue(clout1 *clout.Clout, nodeTemplate *clout.Vertex, input map[string]interface{}) (map[string]interface{}, *clout.Vertex) {
	var val map[string]interface{}
	var node *clout.Vertex

	node = nodeTemplate
	if input["functionCall"] != nil {
		fn := input["functionCall"].(map[string]interface{})
		name := fn["name"].(string)
		args := fn["arguments"].([]interface{})
		if name == "get_property" {
			val, node = getProperty(args, nodeTemplate)
		} else if name == "get_attribute" {
			val, node = getAttribute(args, nodeTemplate)
		} else if name == "get_input" {
			val = getInput(args, nodeTemplate, clout1)
		} else if name == "get_artifact" {
			val = getArtifact(args, nodeTemplate)
		} else {
			fn1 := getJSFunction(clout1, name)
			fmt.Println(fn1)
		}

	} /*else if input["value"] != nil {
		val = input["value"].(string)
	} */
	return val, node
}

func getProperty(args []interface{}, nodeTemplate *clout.Vertex) (map[string]interface{}, *clout.Vertex) {
	if len(args) >= 2 {
		var arg map[string]interface{}
		var props map[string]interface{}
		var node *clout.Vertex
		var propNode *clout.Vertex

		model := args[0].(map[string]interface{})
		modelName := model["value"].(string)

		if modelName == "SELF" {
			node = nodeTemplate
		} else {
			node = getVertexByName(modelName, "nodeTemplate")
		}

		var prop map[string]interface{}

		if node != nil {

			if len(args) == 2 {
				arg = args[1].(map[string]interface{})
				props = node.Properties["properties"].(map[string]interface{})
				name := arg["value"].(string)
				if props[name] != nil {
					prop = props[name].(map[string]interface{})
				}
				propNode = node
			} else {
				prop, propNode = getValueFromCapability(args, node, "property")
				if prop == nil {
					prop, propNode = getValueFromRequirement(args, node, "property")
				}
			}

			return prop, propNode
		}

	}
	return nil, nil
}

func getAttribute(args []interface{}, nodeTemplate *clout.Vertex) (map[string]interface{}, *clout.Vertex) {
	if len(args) >= 2 {
		var arg map[string]interface{}
		var attrs map[string]interface{}
		var node *clout.Vertex
		var attrNode *clout.Vertex

		model := args[0].(map[string]interface{})
		modelName := model["value"].(string)

		if modelName == "SELF" {
			node = nodeTemplate
		} else {
			node = getVertexByName(modelName, "nodeTemplate")
		}

		var attr map[string]interface{}

		if node != nil {
			if len(args) == 2 {
				arg = args[1].(map[string]interface{})
				attrs = node.Properties["attributes"].(map[string]interface{})
				name := arg["value"].(string)
				if attrs[name] != nil {
					attr = attrs[name].(map[string]interface{})
				}
				attrNode = node
			} else {
				attr, attrNode = getValueFromCapability(args, node, "attribute")
				if attr == nil {
					attr, attrNode = getValueFromRequirement(args, node, "attribute")
				}
			}

			return attr, attrNode
		}

	}
	return nil, nil
}

func getInput(args []interface{}, nodeTemplate *clout.Vertex, clout1 *clout.Clout) map[string]interface{} {
	if len(args) >= 1 {
		arg := args[0].(map[string]interface{})
		cloutProps := clout1.Properties["tosca"].(map[string]interface{})
		inputs := cloutProps["inputs"].(map[string]interface{})
		name := arg["value"].(string)
		input := inputs[name].(map[string]interface{})
		return input

	}
	return nil
}

func getArtifact(args []interface{}, nodeTemplate *clout.Vertex) map[string]interface{} {
	if len(args) >= 2 {
		var arg map[string]interface{}
		var artifacts map[string]interface{}
		var node *clout.Vertex

		model := args[0].(map[string]interface{})
		modelName := model["value"].(string)

		if modelName == "SELF" {
			node = nodeTemplate
		} else {
			node = getVertexByName(modelName, "nodeTemplate")
		}

		var artifact map[string]interface{}

		if node != nil {

			if len(args) == 2 {
				arg = args[1].(map[string]interface{})
				artifacts = node.Properties["artifacts"].(map[string]interface{})
				name := arg["value"].(string)
				if artifacts[name] != nil {
					artifact = artifacts[name].(map[string]interface{})
				}
			} else {
				// to complete later; get_artifact with more than 2 arguments
			}

			return artifact

		}

	}
	return nil
}

func getJSFunction(clout1 *clout.Clout, name string) interface{} {
	metadata := clout1.Metadata["puccini-js"].(map[string]interface{})
	return metadata[name]
}

func getValueFromCapability(args []interface{}, node *clout.Vertex, dataType string) (map[string]interface{}, *clout.Vertex) {
	cap := args[1].(map[string]interface{})
	capname := cap["value"].(string)
	capabilities := node.Properties["capabilities"].(map[string]interface{})
	if capabilities[capname] != nil {
		//capability := capabilities[capname].(*clout.Capability)
		capability := capabilities[capname].(map[string]interface{})
		if len(args) == 3 {
			arg := args[2].(map[string]interface{})
			propName := arg["value"].(string)
			if dataType == "property" {
				props := capability["properties"].(map[string]interface{})
				if props[propName] != nil {
					return props[propName].(map[string]interface{}), node
				}
			} else if dataType == "attribute" {
				attrs := capability["attributes"].(map[string]interface{})
				if attrs[propName] != nil {
					return attrs[propName].(map[string]interface{}), node
				}
			}
		} else {
			//to do later
		}
	}
	return nil, nil

}

func getValueFromRequirement(args []interface{}, node *clout.Vertex, dataType string) (map[string]interface{}, *clout.Vertex) {

	var targetNode1 *clout.Vertex
	var targetNode2 *clout.Vertex

	req := args[1].(map[string]interface{})
	reqname := req["value"].(string)
	requirement := getRequirement(node, reqname)
	if requirement != nil {
		if requirement["nodeTemplateName"].(string) != "" {
			targetNode1 = getVertexByName(requirement["nodeTemplateName"].(string), "nodeTemplate")
		} else if requirement["nodeTypeName"].(string) != "" {
			targetNode1 = getVertexByType(requirement["nodeTypeName"].(string), "nodeTemplate")
		}
		if targetNode1 != nil {
			if len(args) == 3 {
				arg := args[2].(map[string]interface{})
				propName := arg["value"].(string)
				props := targetNode1.Properties["properties"].(map[string]interface{})
				if props[propName] != nil {
					return props[propName].(map[string]interface{}), targetNode1
				} else {
					attrs := targetNode1.Properties["attributes"].(map[string]interface{})
					if attrs[propName] != nil {
						return attrs[propName].(map[string]interface{}), targetNode1
					}
				}
			} else if len(args) == 4 {
				arg := args[2].(map[string]interface{})
				argname := arg["value"].(string)
				capabilities := targetNode1.Properties["capabilities"].(map[string]interface{})
				if capabilities[argname] != nil {
					//capability := capabilities[argname].(*clout.Capability)
					capability := capabilities[argname].(map[string]interface{})
					arg := args[3].(map[string]interface{})
					propName := arg["value"].(string)
					if dataType == "property" {
						//props := capability.Properties
						props := capability["properties"].(map[string]interface{})
						if props[propName] != nil {
							return props[propName].(map[string]interface{}), targetNode1
						}
					} else if dataType == "attribute" {
						//attrs := capability.Attributes
						attrs := capability["attributes"].(map[string]interface{})
						if attrs[propName] != nil {
							return attrs[propName].(map[string]interface{}), targetNode1
						}
					}
				} else {
					req1 := getRequirement(targetNode1, argname)
					if req1 != nil {
						if req1["nodeTemplateName"].(string) != "" {
							targetNode2 = getVertexByName(req1["nodeTemplateName"].(string), "nodeTemplate")
						} else if req1["nodeTypeName"].(string) != "" {
							targetNode2 = getVertexByType(req1["nodeTypeName"].(string), "nodeTemplate")
						}
						arg := args[3].(map[string]interface{})
						propName := arg["value"].(string)
						if dataType == "property" {
							//props := targetNode2.Properties
							props := targetNode2.Properties["properties"].(map[string]interface{})
							if props[propName] != nil {
								return props[propName].(map[string]interface{}), targetNode2
							}
						} else if dataType == "attribute" {
							attrs := targetNode2.Properties["attributes"].(map[string]interface{})
							if attrs[propName] != nil {
								return attrs[propName].(map[string]interface{}), targetNode2
							}
						}
					}
				}
			}
		}

	}

	return nil, nil

}

func getRequirement(nodeTemplate *clout.Vertex, name string) map[string]interface{} {
	reqs := nodeTemplate.Properties["requirements"].([]interface{})

	for _, rq := range reqs {
		req := rq.(map[string]interface{})
		if req["name"] == name {
			return req
		}
	}

	return nil
}

func updateNodeAttributeInClout(resultMap map[string]interface{}, outputs map[string]interface{}) {

	if resultMap != nil {
		for k, v := range outputs {
			data := v.(map[string]interface{})
			attrName := data["attributeName"].(string)
			tmplName := data["nodeTemplateName"].(string)
			tmpl := getVertexByName(tmplName, "nodeTemplate")
			if tmpl != nil {
				attrs := tmpl.Properties["attributes"].(map[string]interface{})
				if attrs[attrName] != nil {
					atr := attrs[attrName].(map[string]interface{})
					atr["value"] = resultMap[k].(string)
					tmpl.Properties["attributes"] = attrs
				} else {
					fmt.Println("Error: Attribute with name " + attrName + " not found!!!!")
					// Add the attributes if not present in the target node
					/*atr := make(ard.Map)
					atr["value"] = resultMap[k].(string)
					attrs[attrName] = atr
					tmpl.Properties["attributes"] = attrs*/
				}

			}
		}
	}

}

func getVertexByName(name string, kind string) *clout.Vertex {
	for _, vertex := range baseClout.Vertexes {
		if vertex.Properties["name"] != nil {
			vname := vertex.Properties["name"].(string)
			if isToscaVertex(vertex, kind) && vname == name {
				return vertex
			}
		}

	}
	return nil
}

func getVertexByType(typeName string, kind string) *clout.Vertex {
	for _, vertex := range baseClout.Vertexes {
		if vertex.Properties["types"] != nil {
			types := vertex.Properties["types"].(map[string]interface{})
			if isToscaVertex(vertex, kind) && types[typeName] != nil {
				return vertex
			}
		}
	}
	return nil
}

func writeOutput(data map[string]interface{}) {
	// Write into a file
	file, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(file))

	//_ = ioutil.WriteFile("fw1_csar_dgraph_wf_int1.json", file, 0644)

}
