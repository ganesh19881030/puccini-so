package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/so/db"
	"github.com/tliron/puccini/url"
)

var resolve bool
var coerce bool
var inputs []string
var inputsUrl string

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringArrayVarP(&inputs, "input", "i", []string{}, "specify an input (name=YAML)")
	execCmd.Flags().StringVarP(&inputsUrl, "inputs", "n", "", "load inputs from a PATH or URL to YAML content")
	execCmd.Flags().StringVarP(&output, "output", "o", "", "output to file or directory (default is stdout)")
	execCmd.Flags().BoolVarP(&resolve, "resolve", "r", true, "resolves the topology (attempts to satisfy all requirements with capabilities")
	execCmd.Flags().BoolVarP(&coerce, "coerce", "c", false, "coerces all values (calls functions and applies constraints)")
}

var execCmd = &cobra.Command{
	Use:   "exec [COMMAND or JavaScript PATH or URL] [[Clout PATH or URL]]",
	Short: "Execute JavaScript in Clout",
	Long:  ``,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fn := args[1]

		var quirks []string
		//var path string
		//if len(args) == 2 {
		//	path = args[1]
		//}

		//clout_, err := ReadClout(path)
		if CloutInstanceExists(name) {
			emsg := fmt.Sprintf("Clout instance with name %s already exists!", name)
			fmt.Println(emsg)
			log.Errorf(emsg)
			return
		}
		urlst, err := url.NewValidURL(name, nil)
		common.FailOnError(err)
		dbc := new(db.DgContext)
		var inputValues ard.Map
		if inputsUrl != "" {
			inputValues = ParseInputsFromUrl(inputsUrl)
		}
		if len(inputs) > 0 {
			inputValues = ParseInputsFromCommandLine(inputs)
		}
		st, ok := dbc.ReadServiceTemplateFromDgraph(urlst, inputValues, quirks)
		var clout *clout.Clout
		if !ok {
			return
		} else {
			clout, err = dbc.Compile(st, urlst, resolve, coerce, output)
			common.FailOnError(err)
		}
		//clout_, err := ReadCloutFromDgraph(name)
		// Try loading JavaScript from Clout
		sourceCode, err := js.GetScriptSourceCode(fn, clout)

		if err != nil {
			// Try loading JavaScript from path or URL
			url_, err := url.NewValidURL(fn, nil)
			common.FailOnError(err)

			sourceCode, err = url.Read(url_)
			common.FailOnError(err)

			err = js.SetScriptSourceCode(fn, js.Cleanup(sourceCode), clout)
			common.FailOnError(err)
		}

		err = Exec(fn, sourceCode, clout)
		common.FailOnError(err)
	},
}

func Exec(name string, sourceCode string, c *clout.Clout) error {
	program, err := js.GetProgram(name, sourceCode)
	if err != nil {
		return err
	}

	jsContext := js.NewContext(name, log, common.Quiet, ardFormat, pretty, output)
	_, runtime := jsContext.NewCloutContext(c)
	_, err = runtime.RunProgram(program)

	return js.UnwrapException(err)
}
