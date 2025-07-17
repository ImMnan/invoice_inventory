/*
Copyright © 2025 github.com/immnan
*/
package cmd

import (
	"os"

	"github.com/immnan/invoice_invoice/cmd/get"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lvs",
	Short: "This is an inventory management system developed for ShriKrishna tech",
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func addSubCommand() {
	rootCmd.AddCommand(get.GetCmd)
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.invoice_invoice.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	addSubCommand()
}
