package apply

import (
	"encoding/json"
	"fmt"

	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var ApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Use get command for listing the resources within Blazemeter",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		approve, _ := cmd.Flags().GetBool("approve")
		if err != nil {
			// Handle error
			return
		}
		if file != "" {
			applyProforma(file, approve)
		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
	ApplyCmd.Flags().Bool("approve", false, "[!] Approve the changes")
}

// Execute adds all child commands to the root command and sets flags appropriately.

func applyProforma(fileName string, confirm bool) {
	if !confirm {
		proformaData, err := pkg.GetProforma(fileName)
		if err != nil {
			panic(fmt.Sprintf("error generating proforma data: %v", err))
		}
		var proformaItems []pkg.Proforma
		json.Unmarshal(proformaData, &proformaItems)

		for _, item := range proformaItems {
			fmt.Printf("  Invoice: %s\n", item.Invoice)
			fmt.Printf("  Customer ID: %s\n", item.For)
			fmt.Printf("  Type: %s\n", item.Type)
			fmt.Printf("  For: %s\n", item.For)
			fmt.Printf("  Date: %s\n", item.Date)
			fmt.Printf("  Product UID: %s\n", item.Product.UID)
			fmt.Printf("  Print: %s\n", item.Product.Print)
			fmt.Printf("  Gen: %s\n", item.Product.Gen)
			fmt.Printf("  GST: %vf\n", item.Product.GST)
			for color, quantities := range item.Product.Color {
				fmt.Printf("  Color: %s, XS/S/M/L/XL/2XL/3XL/4XL: %v\n", color, quantities)
			}
			fmt.Printf("  Total: %v\n---\n", item.Product.Total)
		}
	}
	if confirm {
		fmt.Println("Changes approved. Applying proforma to the database...")
		if err := pkg.ApplyProforma(fileName); err != nil {
			panic(fmt.Sprintf("Error applying proforma: %v", err))
		}
	}
}
