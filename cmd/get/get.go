package get

import (
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Use get command for listing the resources within Blazemeter",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {

}
