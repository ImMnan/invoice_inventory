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
		file, _ := cmd.Flags().GetString("file")
		approve, _ := cmd.Flags().GetBool("approve")
		formatCsv, _ := cmd.Flags().GetBool("csv")
		month, _ := cmd.Flags().GetInt("month")
		colorFilter, _ := cmd.Flags().GetString("color")
		if file != "" {
			applyStkInvoice(file, approve, formatCsv, month, colorFilter)
		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
	ApplyCmd.Flags().Bool("approve", false, "[!] Approve the changes")
	ApplyCmd.Flags().Bool("csv", false, "Output in CSV format")
	ApplyCmd.Flags().IntP("month", "m", 0, "Month to apply changes for")
	ApplyCmd.Flags().StringP("color", "c", "", "Filter by color (shows count without --approve)")
}

// Execute adds all child commands to the root command and sets flags appropriately.

func applyStkInvoice(fileName string, confirm, formatCsv bool, month int, colorFilter string) {

	inventoryDB, customerDB, invoiceDB, productDB, err := get.ConfigData(month)
	if err != nil {
		fmt.Printf("Error fetching config data for month %d: %v\n", month, err)
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
		case stockUpdate.SaleEntries != nil && stockUpdate.PurchaseEntries == nil:
			if colorFilter != "" {
				fmt.Printf("\n[!] Filtering by color: %s\n", colorFilter)
			} else {
				fmt.Println("\n[!] Check the sales data correctly before processing the invoice")
			}

			saleLint := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(saleLint, "\n\nPRODUCT ID\tINVOICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tQTY\tTOTAL")
			fmt.Fprintln(saleLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
			// Group SaleEntries by Invoice ID
			invoiceGroups := make(map[string][]pkg.Proforma)
			for _, item := range stockUpdate.SaleEntries {
				invoiceGroups[item.Invoice] = append(invoiceGroups[item.Invoice], item)
			}

			var grandTotal, grandQty int
			var grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X int
			matchedInvoices := 0

			for invoiceId, items := range invoiceGroups {
				var total int
				var qtyTotal int
				var xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total int
				invoiceHasColor := false

				for _, item := range items {
					for _, p := range item.Product {
						for color, quantities := range p.Color {
							// Apply color filter if specified
							if colorFilter != "" && color != colorFilter {
								continue
							}
							invoiceHasColor = true

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
				}

				if invoiceHasColor || colorFilter == "" {
					if invoiceHasColor {
						matchedInvoices++
					}
					fmt.Fprintln(saleLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
					fmt.Fprintf(saleLint, "INVOICE TOTAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal, total)

					grandTotal += total
					grandQty += qtyTotal
					grandXS += xsTotal
					grandS += sTotal
					grandM += mTotal
					grandL += lTotal
					grandXL += xlTotal
					grand2X += x2Total
					grand3X += x3Total
					grand4X += x4Total
				}
			}

			fmt.Fprintln(saleLint, "==========\t=====\t=====\t=====\t==\t==\t==\t==\t==\t==\t==\t==\t==\t====")
			fmt.Fprintf(saleLint, "GRAND TOTAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X, grandQty, grandTotal)
			saleLint.Flush()

			if colorFilter != "" {
				fmt.Printf("\n[*] Found %d invoices with color '%s'\n", matchedInvoices, colorFilter)
				fmt.Printf("[*] Total quantity for color '%s': %d pieces\n", colorFilter, grandQty)
				fmt.Printf("[*] Total amount for color '%s': %d\n", colorFilter, grandTotal)
			}
			fmt.Println("\n[*] The above invoice does not include TAX, the final invoice may look different.")

			// Calculate and display remaining stock after sales
			fmt.Println("\n[*] REMAINING STOCK AFTER SALES:")
			err := displayRemainingStock(existData, stockUpdate, true, colorFilter)
			if err != nil {
				fmt.Printf("Error calculating remaining stock: %v\n", err)
			}

			fmt.Println("\n\n\n'lvs' Copyright (C) 2025  SHRIKRISHNA TECH")

		case stockUpdate.PurchaseEntries != nil && stockUpdate.SaleEntries == nil:
			fmt.Println("\n[!] Check the purchase data correctly before processing the invoice")

			PurchaseLint := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(PurchaseLint, "\n\nPRODUCT ID\tINVOICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tQTY")
			fmt.Fprintln(PurchaseLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--")
			// Group SaleEntries by Invoice ID
			invoiceGroups := make(map[string][]pkg.Proforma)
			for _, item := range stockUpdate.PurchaseEntries {
				invoiceGroups[item.Invoice] = append(invoiceGroups[item.Invoice], item)
			}

			for invoiceId, items := range invoiceGroups {
				var qtyTotal int
				var xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total int
				for _, item := range items {
					for _, p := range item.Product {
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
							line += fmt.Sprintf("\t%d", p.Quantity)
							fmt.Fprintln(PurchaseLint, line)
							qtyTotal += p.Quantity
						}
					}
				}
				fmt.Fprintln(PurchaseLint, "----------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t---")
				fmt.Fprintf(PurchaseLint, "FINAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal)
			}
			PurchaseLint.Flush()

			// Calculate and display remaining stock after purchases
			fmt.Println("\n[*] REMAINING STOCK AFTER PURCHASES:")
			err := displayRemainingStock(existData, stockUpdate, false, "")
			if err != nil {
				fmt.Printf("Error calculating remaining stock: %v\n", err)
			}

			fmt.Println("\n\n\n'lvs' Copyright (C) 2025  SHRIKRISHNA TECH")

		default:
			fmt.Println("No stock updates found, nothing to apply.")
			return
		}
	} else if confirm {
		//fmt.Printf("Debug: %v\n%v", existData, &stockUpdate)
		err := existData.UpdateInventoryFromStockUpdate(&stockUpdate)
		if err != nil {
			fmt.Printf("failed to update inventory from stock update: %v", err)
			return
		}
		invoiceIDs, err := existData.ProcessInvoiceWithStockData(&ProductSlice)
		if err != nil {
			fmt.Printf("failed to make invoice with stock data: %v", err)
			return
		}

		if err := get.PrintInvoice(formatCsv, invoiceIDs, month); err != nil {
			fmt.Printf("failed to print invoice: %v", err)
		}
	} else {
		fmt.Println("Changes not applied. Use --approve to confirm.")
	}

}

// displayRemainingStock calculates and displays the remaining stock after the proposed operation
func displayRemainingStock(existData *pkg.JsLocalDB, stockUpdate pkg.StockUpdate, isSale bool, colorFilter string) error {
	// Load current inventory using existing pkg method
	allEntries, currentStock, err := existData.GetExistingStock()
	if err != nil {
		return fmt.Errorf("failed to load existing inventory: %w", err)
	}

	// Create a copy of current stock for calculation
	remainingStock := make(map[string]map[string][]int)
	for productID, colors := range currentStock {
		remainingStock[productID] = make(map[string][]int)
		for color, quantities := range colors {
			remainingStock[productID][color] = make([]int, len(quantities))
			copy(remainingStock[productID][color], quantities)
		}
	}

	// Apply the stock changes
	var entries []pkg.Proforma
	if isSale {
		entries = stockUpdate.SaleEntries
	} else {
		entries = stockUpdate.PurchaseEntries
	}

	// Process each entry and update remaining stock
	for _, item := range entries {
		for _, product := range item.Product {
			productID := product.ProductID

			// Skip JOB_ products from calculations
			if len(productID) >= 4 && productID[:4] == "JOB_" {
				continue
			}

			// Initialize product in remaining stock if it doesn't exist
			if remainingStock[productID] == nil {
				remainingStock[productID] = make(map[string][]int)
			}

			for color, quantities := range product.Color {
				// Apply color filter if specified and this is a sale
				if isSale && colorFilter != "" && color != colorFilter {
					continue
				}

				// Initialize color if it doesn't exist
				if remainingStock[productID][color] == nil {
					remainingStock[productID][color] = make([]int, 8) // Initialize with zeros
				}

				// Ensure we have 8 sizes
				if len(remainingStock[productID][color]) < 8 {
					padded := make([]int, 8)
					copy(padded, remainingStock[productID][color])
					remainingStock[productID][color] = padded
				}

				// Apply the operation (subtract for sales, add for purchases)
				for i, qty := range quantities {
					if i < len(remainingStock[productID][color]) {
						if isSale {
							remainingStock[productID][color][i] -= qty
						} else {
							remainingStock[productID][color][i] += qty
						}
					}
				}
			}
		}
	}

	// Display the remaining stock table
	remainingLint := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(remainingLint, "PRODUCT ID\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTOTAL")
	fmt.Fprintln(remainingLint, "----------\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t-----")

	var grandTotal int
	var grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X int

	// Get product details for print information
	productDetails := make(map[string]string) // productID -> print
	for _, entry := range allEntries {
		for _, prod := range entry.Product {
			if productDetails[prod.ProductID] == "" {
				productDetails[prod.ProductID] = prod.Print
			}
		}
	}

	// Display remaining stock for each product/color combination
	for productID, colors := range remainingStock {
		// Skip JOB_ products (JOB_DTF type) from the output
		if len(productID) >= 4 && productID[:4] == "JOB_" {
			continue
		}

		for color, quantities := range colors {
			// Apply color filter if specified and this is a sale
			if isSale && colorFilter != "" && color != colorFilter {
				continue
			}

			// Calculate total for this row
			rowTotal := 0
			for _, qty := range quantities {
				rowTotal += qty
			}

			// Only show rows that have some quantity (positive or negative)
			hasQuantity := false
			for _, qty := range quantities {
				if qty != 0 {
					hasQuantity = true
					break
				}
			}

			if hasQuantity || rowTotal != 0 {
				print := productDetails[productID]
				line := fmt.Sprintf("%s\t%s\t%s", productID, print, color)

				// Add size quantities
				for i, qty := range quantities {
					line += fmt.Sprintf("\t%d", qty)
					switch i {
					case 0:
						grandXS += qty
					case 1:
						grandS += qty
					case 2:
						grandM += qty
					case 3:
						grandL += qty
					case 4:
						grandXL += qty
					case 5:
						grand2X += qty
					case 6:
						grand3X += qty
					case 7:
						grand4X += qty
					}
				}
				line += fmt.Sprintf("\t%d", rowTotal)
				fmt.Fprintln(remainingLint, line)
				grandTotal += rowTotal
			}
		}
	}

	fmt.Fprintln(remainingLint, "==========\t=====\t=====\t==\t==\t==\t==\t==\t==\t==\t==\t=====")
	fmt.Fprintf(remainingLint, "GRAND TOTAL\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X, grandTotal)
	remainingLint.Flush()

	if isSale && colorFilter != "" {
		fmt.Printf("\n[*] Remaining stock for color '%s': %d pieces\n", colorFilter, grandTotal)
	}

	// Show warning for negative stock
	hasNegative := false
	for _, colors := range remainingStock {
		for _, quantities := range colors {
			for _, qty := range quantities {
				if qty < 0 {
					hasNegative = true
					break
				}
			}
			if hasNegative {
				break
			}
		}
		if hasNegative {
			break
		}
	}

	if hasNegative {
		fmt.Println("\n[!] WARNING: Some items will have negative stock (indicated by minus values)")
	}

	return nil
}
