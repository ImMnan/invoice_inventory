package apply

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/pkg"
)

func displayRemainingStock(existData *pkg.JsLocalDB, stockUpdate pkg.StockUpdate, isSale bool, colorFilter string, formatCsv bool) error {
	_, currentStock, err := existData.GetExistingStock()
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

	var entries []pkg.Proforma
	if isSale {
		entries = stockUpdate.SaleEntries
	} else {
		entries = stockUpdate.PurchaseEntries
	}

	// Track which product+color pairs are impacted
	type productColor struct{ productID, color string }
	impacted := make(map[productColor]bool)

	for _, item := range entries {
		for _, product := range item.Product {
			productID := product.ProductID
			if (len(productID) >= 4 && productID[:4] == "JOB_") || (len(productID) >= 8 && productID[:8] == "TRADE-ML") {
				continue
			}
			if remainingStock[productID] == nil {
				remainingStock[productID] = make(map[string][]int)
			}
			for color, quantities := range product.Color {
				if isSale && colorFilter != "" && color != colorFilter {
					continue
				}
				if remainingStock[productID][color] == nil {
					remainingStock[productID][color] = make([]int, 8)
				}
				if len(remainingStock[productID][color]) < 8 {
					padded := make([]int, 8)
					copy(padded, remainingStock[productID][color])
					remainingStock[productID][color] = padded
				}
				for i, qty := range quantities {
					if i < len(remainingStock[productID][color]) {
						if isSale {
							remainingStock[productID][color][i] -= qty
						} else {
							remainingStock[productID][color][i] += qty
						}
					}
				}
				impacted[productColor{productID, color}] = true
			}
		}
	}

	var grandTotal int
	var grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X int

	if formatCsv {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"PRODUCT ID", "COLOR", "XS", "S", "M", "L", "XL", "2X", "3X", "4X", "TOTAL"})
		for pc := range impacted {
			padded := make([]int, 8)
			copy(padded, remainingStock[pc.productID][pc.color])
			rowTotal := 0
			for i, q := range padded {
				rowTotal += q
				switch i {
				case 0:
					grandXS += q
				case 1:
					grandS += q
				case 2:
					grandM += q
				case 3:
					grandL += q
				case 4:
					grandXL += q
				case 5:
					grand2X += q
				case 6:
					grand3X += q
				case 7:
					grand4X += q
				}
			}
			grandTotal += rowTotal
			row := []string{pc.productID, pc.color}
			for _, q := range padded {
				row = append(row, strconv.Itoa(q))
			}
			row = append(row, strconv.Itoa(rowTotal))
			_ = w.Write(row)
		}
		_ = w.Write([]string{"GRAND TOTAL", "",
			strconv.Itoa(grandXS), strconv.Itoa(grandS), strconv.Itoa(grandM), strconv.Itoa(grandL),
			strconv.Itoa(grandXL), strconv.Itoa(grand2X), strconv.Itoa(grand3X), strconv.Itoa(grand4X),
			strconv.Itoa(grandTotal),
		})
		w.Flush()
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "PRODUCT ID\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTOTAL")
		fmt.Fprintln(tw, "----------\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t-----")
		for pc := range impacted {
			padded := make([]int, 8)
			copy(padded, remainingStock[pc.productID][pc.color])
			rowTotal := 0
			for i, q := range padded {
				rowTotal += q
				switch i {
				case 0:
					grandXS += q
				case 1:
					grandS += q
				case 2:
					grandM += q
				case 3:
					grandL += q
				case 4:
					grandXL += q
				case 5:
					grand2X += q
				case 6:
					grand3X += q
				case 7:
					grand4X += q
				}
			}
			grandTotal += rowTotal
			line := fmt.Sprintf("%s\t%s", pc.productID, pc.color)
			for _, q := range padded {
				line += fmt.Sprintf("\t%d", q)
			}
			line += fmt.Sprintf("\t%d", rowTotal)
			fmt.Fprintln(tw, line)
		}
		fmt.Fprintln(tw, "==========\t=====\t==\t==\t==\t==\t==\t==\t==\t==\t=====")
		fmt.Fprintf(tw, "GRAND TOTAL\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			grandXS, grandS, grandM, grandL, grandXL, grand2X, grand3X, grand4X, grandTotal)
		tw.Flush()
	}

	if isSale && colorFilter != "" {
		fmt.Printf("\n[*] Remaining stock for color '%s': %d pieces\n", colorFilter, grandTotal)
	}

	return nil
}
