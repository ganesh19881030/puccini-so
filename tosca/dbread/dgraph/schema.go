package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/tliron/puccini/common"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
)

// SchemaContent defines the contents of a predicate schema
type SchemaContent struct {
	Predicate string `json:"predicate"`
	Type      string `json:"type"`
	Index     bool   `json:"index"`
}

// SchemaType defines the slice of predicate schama contents
type SchemaType struct {
	Schema []SchemaContent `json:"schema"`
}

// SaveToscaSchema saves the TOSCA schema in Dgraph
func SaveToscaSchema(ctxt context.Context, dgraphClient *dgo.Dgraph) error {

	var err error
	op := &api.Operation{}
	var dat []byte
	dat, err = ioutil.ReadFile(common.SoConfig.Dgraph.SchemaFilePath)
	if err == nil {
		op.Schema = string(dat)
		err = dgraphClient.Alter(ctxt, op)
	}

	return err
}

// IsSchemaDefined checks if schema predicates are defined in Dgraph
func IsSchemaDefined(ctxt context.Context, dgraphClient *dgo.Dgraph) (bool, error) {
	var defined bool
	var err error

	predlist := []string{"name", "url", "namespace", "description", "nodetemplates"}

	var preds string

	for ind, val := range predlist {
		preds = preds + val
		if ind < len(predlist)-1 {
			preds = preds + ", "
		}
	}
	query := `schema(pred: [%s]) {
		type
		index
	  }`

	req := &api.Request{
		Query: fmt.Sprintf(query, preds),
	}
	txn := dgraphClient.NewTxn()

	if resp, err := txn.Do(ctxt, req); err == nil {
		fmt.Println("json: ", string(resp.GetJson()))
		var schema SchemaType
		json.Unmarshal(resp.GetJson(), &schema)

		for _, pred := range predlist {
			defined = false
			for _, val := range schema.Schema {
				fmt.Println("predicate:", val.Predicate, ",  type:", val.Type, ", index:", val.Index)
				if pred == val.Predicate && len(val.Type) > 0 {
					defined = true
					break
				}
			}
			if !defined {
				break
			}
		}

	}

	txn.Commit(ctxt)

	return defined, err
}
