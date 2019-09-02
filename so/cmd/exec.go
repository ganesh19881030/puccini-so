package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/js"
	"github.com/tliron/puccini/url"
)

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&output, "output", "o", "", "output to file or directory (default is stdout)")
}

var execCmd = &cobra.Command{
	Use:   "exec [COMMAND or JavaScript PATH or URL] [[Clout PATH or URL]]",
	Short: "Execute JavaScript in Clout",
	Long:  ``,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fn := args[1]

		//var path string
		//if len(args) == 2 {
		//	path = args[1]
		//}

		//clout_, err := ReadClout(path)
		clout_, err := ReadCloutFromDgraph(name)
		common.FailOnError(err)

		// Try loading JavaScript from Clout
		sourceCode, err := js.GetScriptSourceCode(fn, clout_)

		if err != nil {
			// Try loading JavaScript from path or URL
			url_, err := url.NewValidURL(fn, nil)
			common.FailOnError(err)

			sourceCode, err = url.Read(url_)
			common.FailOnError(err)

			err = js.SetScriptSourceCode(fn, js.Cleanup(sourceCode), clout_)
			common.FailOnError(err)
		}

		err = Exec(fn, sourceCode, clout_)
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
