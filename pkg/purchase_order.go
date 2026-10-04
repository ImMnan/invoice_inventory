package pkg

import "strings"

func (product *ProductSlice) addPurchase() (map[string]map[string][]int, []Purchase, error) {

	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	var purchaseEntries []Purchase

	for _, productItem := range *product {
		// Only process purchase-invoice entries, skip proforma entries
		if productItem.Type != "proforma" && productItem.Type == "purchase-invoice" && productItem.Type != "job" {
			for _, prod := range productItem.Product {
				stockKey := StockKey(prod.ProductID, prod.Print)
				purchaseEntry := Purchase{
					UUID:    productItem.UUID,
					Type:    "purchase",
					From:    productItem.From, // Copy vendor ID
					Invoice: productItem.Invoice,
					Date:    productItem.Date,
					Product: []ProductStruct{{
						ProductID: prod.ProductID,
						Print:     NormalizePrint(prod.Print),
						Gen:       prod.Gen,
						Color:     make(map[string][]int),
						Quantity:  prod.Quantity,
						Total:     prod.Total,
					}},
				}
				for color, quantities := range prod.Color {
					purchaseEntry.Product[0].Color[strings.ToLower(strings.TrimSpace(color))] = quantities
				}
				purchaseEntries = append(purchaseEntries, purchaseEntry)
				if stockUpdates[stockKey] == nil {
					stockUpdates[stockKey] = make(map[string][]int)
				}
				for color, quantities := range prod.Color {
					colorKey := strings.ToLower(strings.TrimSpace(color))
					if existing, exists := stockUpdates[stockKey][colorKey]; exists {
						for i, qty := range quantities {
							if i < len(existing) {
								stockUpdates[stockKey][colorKey][i] += qty
							}
						}
					} else {
						stockUpdates[stockKey][colorKey] = make([]int, len(quantities))
						copy(stockUpdates[stockKey][colorKey], quantities)
					}
				}
			}
		}
	}
	return stockUpdates, purchaseEntries, nil
}
