/*
Copyright © 2025 github.com/immnan
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/immnan/invoice_invoice/cmd/apply"
	"github.com/immnan/invoice_invoice/cmd/delete"
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
		help, _ := cmd.Flags().GetBool("help")
		license, _ := cmd.Flags().GetBool("license")
		version, _ := cmd.Flags().GetBool("version")
		switch {
		case help:
			cmd.Help()
		case license:
			printLicense()
		case version:
			versionInfo()
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
	rootCmd.AddCommand(apply.ApplyCmd)
	rootCmd.AddCommand(delete.DeleteCmd)
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.invoice_invoice.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	rootCmd.Flags().BoolP("license", "l", false, "Show license information")
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
	addSubCommand()
}

func printLicense() {
	fmt.Println("\n---")

	fmt.Println(

		`    This is lvs, a tool to manage inventory, sales, purchases and invoices
    at Shrikrishna Tech

    Copyright (C) 2025  SHRIKRISHNA TECH

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.`)

	fmt.Println(`    'lvs'  Copyright (C) 2025  SHRIKRISHNA TECH
    This program comes with ABSOLUTELY NO WARRANTY; for details type.
    This is free software, and you are welcome to redistribute it
    under certain conditions; type "lvs --license" for details.`)

	fmt.Println("\n---")
}

func versionInfo() {
	fmt.Printf("\n     version: %s\n", "0.4.1-alpha\n")
	fmt.Println(`    'lvs'  Copyright (C) 2025  SHRIKRISHNA TECH
    This program comes with ABSOLUTELY NO WARRANTY; for details type.
    This is free software, and you are welcome to redistribute it
    under certain conditions; type "lvs --license" for details.`)

	fmt.Println("\n---")
}
