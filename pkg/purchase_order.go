package pkg

import "strings"

func (product *ProductSlice) addPurchase() (map[string]map[string][]int, []Purchase, error) {

	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	var purchaseEntries []Purchase

	for _, productItem := range *product {
		// Only process purchase-invoice entries, skip proforma entries
		if productItem.Type != "purchase-invoice" {
			continue
		}

		purchaseEntry := Purchase{
			UUID:    productItem.UUID,
			Type:    productItem.Type,
			From:    productItem.From, // Copy vendor ID
			Invoice: productItem.Invoice,
			Date:    productItem.Date,
			Product: ProductStruct{
				UID:   productItem.Product.UID,
				Gen:   productItem.Product.Gen,
				Color: make(map[string][]int),
				Total: productItem.Product.Total,
			},
		}
		//		purchaseEntries = append(purchaseEntries, purchaseEntry)
		for color, quantities := range productItem.Product.Color {
			purchaseEntry.Product.Color[strings.ToLower(color)] = quantities
		}

		purchaseEntries = append(purchaseEntries, purchaseEntry)

		if stockUpdates[productItem.Product.UID] == nil {
			stockUpdates[productItem.Product.UID] = make(map[string][]int)
		}

		for color, quantities := range productItem.Product.Color {
			colorKey := strings.ToLower(color)
			if existing, exists := stockUpdates[productItem.Product.UID][colorKey]; exists {
				for i, qty := range quantities {
					if i < len(existing) {
						stockUpdates[productItem.Product.UID][colorKey][i] += qty
					}
				}

			} else {
				//initialize with current quantities
				stockUpdates[productItem.Product.UID][colorKey] = make([]int, len(quantities))
				copy(stockUpdates[productItem.Product.UID][colorKey], quantities)
			}

		}

	}
	return stockUpdates, purchaseEntries, nil
}
