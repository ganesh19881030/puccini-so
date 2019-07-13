package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"encoding/json"

	"github.com/tliron/puccini/common"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

// CancelFunc - declare cancel function for dgraph client connection
type CancelFunc func()

// GetDgraphClient - Fetch a Dgraph Go client connected to a Dgraph db
func GetDgraphClient(dburl string, isEnterprise bool) (*dgo.Dgraph, CancelFunc) {
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

// DgraphDropAll - clear Dgraph database
func DgraphDropAll(dburl string) {
	dg, cancel := GetDgraphClient(dburl, false)
	defer cancel()
	op := api.Operation{DropAll: true}
	ctx := context.Background()
	if err := dg.Alter(ctx, &op); err != nil {
		log.Fatal(err)
	}
	// Output:
}

// SaveCloutGraph - Save the clout dgraph set in Dgraph db
func SaveCloutGraph(dgraphSet *DgraphSet, dburl string) {
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