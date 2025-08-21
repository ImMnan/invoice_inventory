package apply

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/cmd/get"
	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// applyCMD represents the get command
var ApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Use get command for applying the resources to the Database",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		approve, _ := cmd.Flags().GetBool("approve")
		formatCsv, _ := cmd.Flags().GetBool("csv")

		if err != nil {
			// Handle error
			return
		}

		if file != "" {
			applyProforma(file, approve, formatCsv)

		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
	ApplyCmd.Flags().Bool("approve", false, "[!] Approve the changes")
	ApplyCmd.Flags().BoolP("csv", "c", false, "Output in CSV format")
}

// Execute adds all child commands to the root command and sets flags appropriately.

func applyProforma(fileName string, confirm, formatCsv bool) {

	inventoryDB, customerDB, invoiceDB, productDB, err := get.ConfigData(0)
	if err != nil {
		fmt.Println("Error fetching config data:", err)
		return
	}
	existData := &pkg.JsLocalDB{
		InventoryFile: inventoryDB,
		CustomerFile:  customerDB,
		InvoiceFile:   invoiceDB,
		ProductFile:   productDB,
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

	if !confirm {
		switch {
		case stockUpdate.SaleEntries != nil:
			fmt.Println("\n[!] Check the data correctly before processing the invoice")

			saleLint := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(saleLint, "\n\nPRODUCT ID\tINVOICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tQTY\tTOTAL")
			fmt.Fprintln(saleLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
			// Group SaleEntries by Invoice ID
			invoiceGroups := make(map[string][]pkg.Proforma)
			for _, item := range stockUpdate.SaleEntries {
				invoiceGroups[item.Invoice] = append(invoiceGroups[item.Invoice], item)
			}

			for invoiceId, items := range invoiceGroups {
				var total int
				var qtyTotal int
				var xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total int
				for _, item := range items {
					p := item.Product
					for color, quantities := range p.Color {
						line := fmt.Sprintf("%s\t%s\t%s\t%s",
							p.ProductID,
							invoiceId,
							p.Print,
							color,
						)
						// Ensure we have 8 sizes: XS, S, M, L, XL, 2X, 3X, 4X
						padded := make([]int, 8)
						copy(padded, quantities)
						xsTotal += padded[0]
						sTotal += padded[1]
						mTotal += padded[2]
						lTotal += padded[3]
						xlTotal += padded[4]
						x2Total += padded[5]
						x3Total += padded[6]
						x4Total += padded[7]
						for _, q := range padded {
							line += fmt.Sprintf("\t%d", q)
						}
						line += fmt.Sprintf("\t%d\t%d", p.Quantity, p.Total)
						fmt.Fprintln(saleLint, line)
						total += p.Total
						qtyTotal += p.Quantity
					}
				}
				fmt.Fprintln(saleLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
				fmt.Fprintf(saleLint, "FINAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal, total)
			}
			saleLint.Flush()
			fmt.Println("\n[*] The above invoice does not include TAX, the final invoice may look different.")
			fmt.Println("\n\n\n'lvs' Copyright (C) 2025  SHRIKRISHNA TECH")

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
		//fmt.Printf("Debug: %v\n%v", existData, &stockUpdate)
		invoiceGroupedData, err := existData.UpdateInventoryFromStockUpdate(&stockUpdate)
		if err != nil {
			fmt.Printf("failed to update inventory from stock update: %v", err)
			return
		}

		//fmt.Printf("Debug: %v\n%v", invoiceGroupedData, &ProductSlice)

		invoiceIDs, err := existData.ProcessInvoiceWithStockData(invoiceGroupedData, &ProductSlice)
		if err != nil {
			fmt.Printf("failed to make invoice with stock data: %v", err)
			return
		}

		if err := get.PrintInvoice(formatCsv, invoiceIDs, 0); err != nil {
			fmt.Printf("failed to print invoice: %v", err)
		}
	} else {
		fmt.Println("Changes not applied. Use --approve to confirm.")
	}

}
