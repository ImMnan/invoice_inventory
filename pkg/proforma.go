package pkg

import (
	"strings"

	"github.com/google/uuid"
)

func (product *ProductSlice) addProforma() (map[string]map[string][]int, []Proforma, error) {
	// Extract customer ID from the first proforma item
	//var customerID string
	//if len(proformaItems) > 0 {
	//	customerID = proformaItems[0].For
	//}

	// Create sale entries and track stock changes
	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities

	var saleEntries []Proforma

	for _, productItem := range *product {
		if productItem.Type == "proforma" && productItem.Type != "purchase-invoice" {
			// Create a sale entry from the proforma item
			saleEntry := Proforma{
				UUID:     uuid.New().String(),
				Type:     "sale",
				For:      productItem.For, // Copy customer ID
				Invoice:  productItem.Invoice,
				Date:     productItem.Date,
				IsPaid:   false,
				Rejected: false,
				Product: ProductStruct{
					UID:   productItem.Product.UID,
					Print: productItem.Product.Print,
					Gen:   productItem.Product.Gen,
					GST:   productItem.Product.GST,
					Color: make(map[string][]int),
					Total: productItem.Product.Total,
				},
			}
			// Copy color data to sale entry (normalize color keys to lowercase)
			for color, quantities := range productItem.Product.Color {
				saleEntry.Product.Color[strings.ToLower(color)] = quantities
			}

			saleEntries = append(saleEntries, saleEntry)
			// Initialize stock updates for this product if not exists
			if stockUpdates[productItem.Product.UID] == nil {
				stockUpdates[productItem.Product.UID] = make(map[string][]int)
			}

			// Track quantities to subtract from stock
			for color, quantities := range productItem.Product.Color {
				colorKey := strings.ToLower(color)
				if existing, exists := stockUpdates[productItem.Product.UID][colorKey]; exists {
					// Add to existing quantities
					for i, qty := range quantities {
						if i < len(existing) {
							existing[i] += qty
						}
					}
				} else {
					// Initialize with current quantities
					stockUpdates[productItem.Product.UID][colorKey] = make([]int, len(quantities))
					copy(stockUpdates[productItem.Product.UID][colorKey], quantities)
				}
			}
		}
	}
	return stockUpdates, saleEntries, nil
}
