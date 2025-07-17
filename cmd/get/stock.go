package get

import "github.com/spf13/cobra"

// rootCmd represents the base command when called without any subcommands
var stockCmd = &cobra.Command{
	Use:   "stock",
	Short: "to get stock details",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		help := cmd.Flag("help")
		if help != nil && help.Value.String() == "true" {
			// Show help information
			cmd.Help()
		}

	},
}

func init() {
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	GetCmd.AddCommand(stockCmd)

}
