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

	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
