package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	//"fmt"
	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/database"
	"github.com/tliron/puccini/tosca/dbread/dgraph"

	//"github.com/gorilla/mux"
	//"github.com/tliron/puccini/clout"
	//"github.com/tliron/puccini/common"
	//"github.com/tliron/puccini/js"
	//"github.com/tliron/puccini/url"
	"github.com/tliron/puccini/ard"
	"google.golang.org/grpc"
	//"log"
	//"net/http"
)

func fetchDbTemplate() (*dgraph.DgraphTemplate, error) {
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)
	dgraphClient, conn, err := database.GetDgraphClient1(dburl)
	dgt := new(dgraph.DgraphTemplate)
	dgt.Ctxt = context.Background()
	dgt.Client = dgraphClient
	dgt.Conn = conn
	return dgt, err
}
func createConnection() *grpc.ClientConn {
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)
	conn, err := grpc.Dial(dburl, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

//func getServiceUID(tname string, sname string) string {
func getServiceUID(tname string) string {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	/*const q = `query all($name: string, $sname: string, $stype: string) {
		all(func: eq(<clout:templateName>, $name)) @filter (eq(<clout:name>, $sname)
			AND eq(<clout:type>, $stype)){
			uid
	    }
	}`*/
	const q = `query all($name: string, $sname: string, $stype: string) {
		all(func: eq(<clout:name>, $name)) {
			uid
	    }
	}`
	//resp, err := txn.QueryWithVars(context.Background(), q,
	//	map[string]string{"$name": tname, "$sname": sname, "$stype": "service"})

	resp, err := txn.QueryWithVars(context.Background(), q,
		map[string]string{"$name": tname})

	if err != nil {
		return ""
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

}

//func getServiceInputs(tname string, sname string) (string, ard.Map) {
func getServiceInputs(tname string) (string, ard.Map, error) {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	/*const q = `query all($name: string, $sname: string, $stype: string) {
		all(func: eq(<clout:templateName>, $name)) @filter (eq(<clout:name>, $sname)
			AND eq(<clout:type>, $stype)){
			uid
			<clout:properties>
	    }
	}`*/
	const q = `query all($name: string) {
		all(func: eq(<clout:name>, $name)) {
			uid
			<clout:properties>
	    }
	}`
	/*resp, err := txn.QueryWithVars(context.Background(), q,
	map[string]string{"$name": tname, "$sname": sname, "$stype": "service"})*/

	resp, err := txn.QueryWithVars(context.Background(), q, map[string]string{"$name": tname})

	if err != nil {
		return "", nil, err
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		if err != nil {
			log.Errorf(err.Error())
			return "", nil, err
		}
	}
	uid := ""
	var props ard.Map

	queryData := result["all"].([]interface{})
	if len(queryData) != 0 {
		for _, t := range queryData {
			cloutMap := t.(map[string]interface{})
			uid = cloutMap["uid"].(string)
			props, err = getPropMap(cloutMap["clout:properties"])
			if err != nil {
				return "", nil, err
			}
		}
	}
	return uid, props["inputs"].(ard.Map), nil

}

func getServiceNode(id string, nodeName string) map[string]interface{} {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	/*const q = `query all($id: string, $nname: string, $ntype: string) {
			all(func: uid($id)) {
	    		<clout:vertex> @filter(eq(<tosca:entity>, "$ntype") AND eq(<tosca:name>, "$nname")){
	      			uid
	      			<tosca:attributes>
	      		}
		    }
		}`*/

	// Query the clout vertex
	const q = `query all($id: string, $nname: string, $ntype: string) {
		all(func: uid($id)) {
    		<clout:vertex> @filter(eq(<tosca:entity>, "$ntype") AND eq(<tosca:name>, "$nname")){
      			uid
      			<tosca:attributes>
      		}
	    }
	}`

	//resp, err := txn.QueryWithVars(context.Background(), q,
	//	map[string]string{"$id": id, "$nname": nodeName, "$ntype": "node"})

	resp, err := txn.QueryWithVars(context.Background(), q,
		map[string]string{"$id": id, "$nname": nodeName, "$ntype": "nodeTemplate"})

	if err != nil {
		return nil
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}
	return result

}

func getNodeAttributes(uid string, nodeName string) ard.Map {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	// Query the clout vertex
	const q = `query all($tname: string, $nname: string) {
		all(func: uid("$uid")) {
    		<clout:vertex> @filter(eq(<tosca:entity>, "nodeTemplate") AND eq(<tosca:name>, "$nname")){
      			<tosca:attributes>
      		}
	    }
	}`

	resp, err := txn.QueryWithVars(context.Background(), q,
		map[string]string{"$uid": uid, "$nname": nodeName})

	if err != nil {
		return nil
	}

	var result map[string]interface{}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		log.Fatal(err)
	}
	return result

}

func updateNodeAttributes(node Node) {
	conn := createConnection()

	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	txn := dgraphClient.NewTxn()
	ctx := context.Background()
	defer txn.Discard(ctx)

	out, err := json.Marshal(node)
	if err != nil {
		log.Fatal(err)
	}

	_, err = txn.Mutate(ctx, &api.Mutation{SetJson: out})
	if err != nil {
		log.Fatal(err)
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
