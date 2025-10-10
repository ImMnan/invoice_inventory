package delete

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteCMD represents the get command
var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Use delete command for removing the resources from the Database",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		deleteItem()
	},
}

func init() {
}

func deleteItem() {
	fmt.Println("delete called")
}
