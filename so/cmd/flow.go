package cmd

import (
	"bytes"
	"fmt"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	flow "github.com/tliron/puccini/wf/components"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	//"os/exec"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var instanceName string
var templateName string
var uid string

type Node struct {
	Uid        string `json:"uid" yaml:"uid"`
	Attributes string `json:"tosca:attributes" yaml:"tosca:attributes"`
}

func CreateWorkflows(clout *clout.Clout) map[string]*flow.Workflow {

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

func ExecuteWorkflow(workflow *flow.Workflow, tname string, sname string) *flow.WfError {
	instanceName = sname
	templateName = tname
	uid = getServiceUID(templateName, instanceName)
	err := workflow.Run()
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
					return CallOperation(nodeTemplates, groups, activity)
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

/*func SetNodeState(params interface{}) *flow.WfError {
	p := params.(map[string]interface{})
	act := p["activity"].(map[string]interface{})
	val := act["setNodeState"].(string)

	nodeTemplates := p["nodeTemplates"].([]*clout.Vertex)
	for _, nodeTemplate := range nodeTemplates {
		nodeAttr := nodeTemplate.Properties["attributes"].(map[string]interface{})
		state := nodeAttr["state"].(map[string]interface{})
		state["value"] = val
	}

	groups := p["groups"].([]*clout.Vertex)
	for _, group := range groups {
		grpAttr := group.Properties["attributes"].(map[string]interface{})
		state := grpAttr["state"].(map[string]interface{})
		state["value"] = val
	}
	fmt.Println(val)
	return nil
}*/

func SetNodeState(nodeTemplates []*clout.Vertex, groups []*clout.Vertex, activity interface{}) *flow.WfError {
	//fmt.Println(instanceName)
	act := activity.(map[string]interface{})
	val := act["setNodeState"].(string)

	for _, nodeTemplate := range nodeTemplates {
		nodeName := nodeTemplate.Properties["name"].(string)
		/*nodeAttr := nodeTemplate.Properties["attributes"].(map[string]interface{})
		state := nodeAttr["state"].(map[string]interface{})
		state["value"] = val*/
		fmt.Println(nodeTemplate.Properties["name"].(string) + " set Node state ===> " + val)
		updateNodeData(nodeName, "state", val)
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

func updateNodeData(nodeName string, key string, val string) {
	result := getServiceNode(uid, nodeName)
	if result == nil || len(result) <= 0 {
		//fmt.Println("Node " + nodeName + " has no attribute called " + key)
		return
	}
	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {
		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			vertexList := cloutMap["clout:vertex"].([]interface{})
			vertex := vertexList[0].(map[string]interface{})
			nodeUid := vertex["uid"].(string)
			attrs := getPropMap(vertex["tosca:attributes"])
			atr := attrs[key].(map[string]interface{})
			atr["value"] = val
			bytes, _ := json.Marshal(attrs)
			node := Node{Uid: nodeUid, Attributes: string(bytes)}
			updateNodeAttributes(node)
			fmt.Println(node)
			fmt.Println(atr)
		}
	}
}

/*func CallOperation(params interface{}) *flow.WfError {
	p := params.(map[string]interface{})
	act := p["activity"].(map[string]interface{})
	callOp := act["callOperation"].(map[string]interface{})
	inter := callOp["interface"].(string)
	operName := callOp["operation"].(string)

	nodeTemplates := p["nodeTemplates"].([]*clout.Vertex)
	for _, nodeTemplate := range nodeTemplates {
		nodeInterface := nodeTemplate.Properties["interfaces"].(map[string]interface{})
		inter := nodeInterface[inter].(map[string]interface{})
		opers := inter["operations"].(map[string]interface{})
		oper := opers[operName].(map[string]interface{})
		fmt.Println(oper["description"].(string))
	}

	groups := p["groups"].([]*clout.Vertex)
	for _, group := range groups {
		grpInterface := group.Properties["interfaces"].(map[string]interface{})
		inter := grpInterface[inter].(map[string]interface{})
		opers := inter["operations"].(map[string]interface{})
		oper := opers[operName].(map[string]interface{})
		fmt.Println(oper["description"].(string))
	}

	return nil
}*/

func CallOperation(nodeTemplates []*clout.Vertex, groups []*clout.Vertex,
	activity interface{}) *flow.WfError {
	act := activity.(map[string]interface{})
	callOp := act["callOperation"].(map[string]interface{})
	inter := callOp["interface"].(string)
	operName := callOp["operation"].(string)

	for _, nodeTemplate := range nodeTemplates {
		nodeName := nodeTemplate.Properties["name"].(string)
		//nodeInterface := nodeTemplate.Properties["interfaces"].(map[string]interface{})
		//inter := nodeInterface[inter].(map[string]interface{})
		//opers := inter["operations"].(map[string]interface{})
		//oper := opers[operName].(map[string]interface{})
		//name := nodeTemplate.Properties["name"].(string)
		//fmt.Println(name + " ===> " + oper["description"].(string))
		//err := exec.Command("C:\\Python37-32\\python.exe", "../test/tic-tac-toe.py").Run()

		//Using Python
		/*cmd := exec.Command("C:\\Python37-32\\python.exe", "../test/remote1.py")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			//return
		}
		fmt.Println("Result: " + out.String())*/

		cmds := make([]string, 0)
		//cmds = append(cmds, "eval `ssh-agent -s`")
		//cmds = append(cmds, "ssh-add ~/.ssh/ohio-key-pair.pem")
		//cmds = append(cmds, "cd /opt/app/bonap")
		//cmds = append(cmds, "sudo /opt/app/bonap/scripts/create_server.sh /opt/app/bonap/ansible/create_"+name+".yml")
		cmds = append(cmds, "sudo /opt/app/bonap/scripts/create_server.sh /opt/app/bonap/ansible/create_instance.yml "+nodeName)
		cmds = append(cmds, "cat /opt/app/bonap/pub-ip.txt")
		//ip := execRemoteCommand(cmds)
		ip := "xyz.xyz.xyz.xyz"
		updateNodeData(nodeName, "public_address", ip)
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

func execRemoteCommand(cmds []string) string {
	//key, err := ssh.ParsePrivateKey([]byte(privateKey))
	user := common.SoConfig.Remote.RemoteUser
	addr := common.SoConfig.Remote.RemoteHost
	port := common.SoConfig.Remote.RemotePort
	pkfile := common.SoConfig.Remote.RemotePubKey
	pkBytes, err := ioutil.ReadFile(pkfile)
	//pkBytes, err := ioutil.ReadFile("../test/ohio-key-pair.pem")
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
	}
	key, err := ssh.ParsePrivateKey(pkBytes)
	if err != nil {
		fmt.Println(err)
		//return "", err
	}
	// Authentication
	//cmd := "df -h"
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
		//return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		fmt.Println(err)
		//return "", err
	}
	defer session.Close()
	var b bytes.Buffer // import "bytes"
	//session.Stderr = &b // get error
	session.Stdout = &b // get output
	// you can also pass what gets input to the stdin, allowing you to pipe
	// content from client to server
	//      session.Stdin = bytes.NewBufferString("My input")

	// Finally, run the command
	cmd := strings.Join(cmds, "; ")
	err = session.Run(cmd)
	//session.Stderr = &b
	//a := session.Stderr
	fmt.Println(b.String())
	output := b.String()
	ss := strings.Split(output, " ")
	ip := strings.TrimSpace(ss[len(ss)-1])
	fmt.Println("IP: " + ip)
	//fmt.Println(a)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return ip
}
