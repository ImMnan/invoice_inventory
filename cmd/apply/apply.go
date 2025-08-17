package apply

import (
	"fmt"

	"github.com/immnan/invoice_invoice/get"
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
		taxed, _ := cmd.Flags().GetBool("taxed")
		csv := false
		csv, _ = cmd.Flags().GetBool("csv")

		if err != nil {
			// Handle error
			return
		}

		if file != "" {
			applyProforma(file, approve, taxed, csv)

		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
	ApplyCmd.Flags().Bool("approve", false, "[!] Approve the changes")
	ApplyCmd.Flags().BoolP("csv", "c", false, "Output in CSV format")
	ApplyCmd.Flags().BoolP("taxed", "t", false, "Create/Apply TAX invoice")
}

// Execute adds all child commands to the root command and sets flags appropriately.

func applyProforma(fileName string, confirm, taxed, csv bool) {
	var format string
	if csv {
		format = "csv"
	} else {
		format = "table"
	}

	var updateData *pkg.FileData
	if fileName != "" {
		updateData = &pkg.FileData{Data: fileName}
	}
	ProductSlice, err := updateData.GetStockUpdate()
	if err != nil {
		fmt.Printf("failed to get stock update: %v", err)
		return
	}

	stockUpdate, err := pkg.MakeStkUpdate(&ProductSlice)
	if err != nil {
		fmt.Printf("failed to make stock update: %v", err)
		return
	}
	existData := &pkg.JsLocalDB{File: "Data/inventory.json"}

	if !confirm {
		switch {
		case stockUpdate.SaleEntries != nil:
			fmt.Println("Proforma stock updates found, processing...")
			for _, entry := range stockUpdate.SaleEntries {
				fmt.Printf("%v\n", entry)
			}
		case stockUpdate.PurchaseEntries != nil:
			fmt.Println("Purchase stock updates found, processing...")
			for _, entry := range stockUpdate.PurchaseEntries {
				fmt.Printf("%v\n", entry)
			}
		default:
			fmt.Println("No stock updates found, nothing to apply.")
			return
		}
	} else if confirm {
		invoiceGroupedData, err := existData.UpdateInventoryFromStockUpdate(&stockUpdate)
		if err != nil {
			fmt.Printf("failed to update inventory from stock update: %v", err)
			return
		}

		invoiceIDs, err := ProductSlice.ProcessInvoiceWithStockData(taxed, format, invoiceGroupedData)
		if err != nil {
			fmt.Printf("failed to make invoice with stock data: %v", err)
			return
		}

		if err := get.PrintInvoice(format, invoiceIDs); err != nil {
			fmt.Printf("failed to print invoice: %v", err)
		}
	} else {
		fmt.Println("Changes not applied. Use --approve to confirm.")
	}
}
