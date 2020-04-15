package main

import (
	//"fmt"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/so/cmd"
	//"os"
)

/*func main() {
	name := os.Args[1]
	wfName := os.Args[2]
	clout, err := cmd.ReadCloutFromDgraph(name)
	if err != nil {
		fmt.Println("error reading clout")
		return
	}
	cmd.ProcessWorkflow(clout, wfName)
}*/

func main() {
	runasService := true

	// read configuration from a file
	common.ReadConfiguration()

	if runasService {
		cmd.HandleRequests()
	} else {
		cmd.Execute()
	}
}

/*func main() {
	cmd.Execute()
}*/
