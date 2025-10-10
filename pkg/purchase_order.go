package pkg

import "strings"

func (product *ProductSlice) addPurchase() (map[string]map[string][]int, []Purchase, error) {

	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	var purchaseEntries []Purchase

	for _, productItem := range *product {
		// Only process purchase-invoice entries, skip proforma entries
		if productItem.Type != "proforma" && productItem.Type == "purchase-invoice" && productItem.Type != "job" {
			for _, prod := range productItem.Product {
				purchaseEntry := Purchase{
					UUID:    productItem.UUID,
					Type:    "purchase",
					From:    productItem.From, // Copy vendor ID
					Invoice: productItem.Invoice,
					Date:    productItem.Date,
					Product: []ProductStruct{{
						ProductID: prod.ProductID,
						Gen:       prod.Gen,
						Color:     make(map[string][]int),
						Quantity:  prod.Quantity,
						Total:     prod.Total,
					}},
				}
				for color, quantities := range prod.Color {
					purchaseEntry.Product[0].Color[strings.ToLower(color)] = quantities
				}
				purchaseEntries = append(purchaseEntries, purchaseEntry)
				if stockUpdates[prod.ProductID] == nil {
					stockUpdates[prod.ProductID] = make(map[string][]int)
				}
				for color, quantities := range prod.Color {
					colorKey := strings.ToLower(color)
					if existing, exists := stockUpdates[prod.ProductID][colorKey]; exists {
						for i, qty := range quantities {
							if i < len(existing) {
								stockUpdates[prod.ProductID][colorKey][i] += qty
							}
						}
					} else {
						stockUpdates[prod.ProductID][colorKey] = make([]int, len(quantities))
						copy(stockUpdates[prod.ProductID][colorKey], quantities)
					}
				}
			}
		}
	}
	return stockUpdates, purchaseEntries, nil
}
