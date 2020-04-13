package dgraph

import (
	"context"

	"github.com/tliron/puccini/common"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
)

type DbTemplate interface {
	ExecQuery(query string) (*api.Response, error)
	ExecMutation(nquad string) (*api.Response, error)
}

type DgraphTemplate struct {
	Ctxt   context.Context
	Client *dgo.Dgraph
}

func (dgt *DgraphTemplate) ExecQuery(query string) (*api.Response, error) {
	req := &api.Request{
		Query: query,
	}

	//Log.Debugf("\nQuery: -->\n%s", query)
	txn := dgt.Client.NewTxn()
	resp, err := txn.Do(dgt.Ctxt, req)
	if err == nil {
		//Log.Debugf("\nResponse JSON: -->\n%s", string(resp.GetJson()))
	} else {
		common.FailOnError(err)
	}

	txn.Commit(dgt.Ctxt)

	return resp, err

}

func (dgt *DgraphTemplate) ExecMutation(nquad string) (*api.Response, error) {

	mu := &api.Mutation{
		SetNquads: []byte(nquad),
	}

	req := &api.Request{
		Mutations: []*api.Mutation{mu},
		//CommitNow: true,
	}

	//Log.Debugf("\nQuery: -->\n%s", query)
	txn := dgt.Client.NewTxn()
	resp, err := txn.Do(dgt.Ctxt, req)
	if err == nil {
		//Log.Debugf("\nResponse JSON: -->\n%s", string(resp.GetJson()))
	} else {
		common.FailOnError(err)
	}

	txn.Commit(dgt.Ctxt)

	return resp, err

}
