package pkg

import "strings"

func (product *ProductSlice) addPurchase() (map[string]map[string][]int, []Purchase, error) {

	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	var purchaseEntries []Purchase

	for _, productItem := range *product {
		// Only process purchase-invoice entries, skip proforma entries
		if productItem.Type != "proforma" && productItem.Type == "purchase-invoice" && productItem.Type != "job" {

			purchaseEntry := Purchase{
				UUID:    productItem.UUID,
				Type:    "purchase",
				From:    productItem.From, // Copy vendor ID
				Invoice: productItem.Invoice,
				Date:    productItem.Date,
				Product: ProductStruct{
					ProductID: productItem.Product.ProductID,
					Gen:       productItem.Product.Gen,
					Color:     make(map[string][]int),
					Quantity:  productItem.Product.Quantity,
					Total:     productItem.Product.Total,
				},
			}
			//		purchaseEntries = append(purchaseEntries, purchaseEntry)
			for color, quantities := range productItem.Product.Color {
				purchaseEntry.Product.Color[strings.ToLower(color)] = quantities
			}

			purchaseEntries = append(purchaseEntries, purchaseEntry)

			if stockUpdates[productItem.Product.ProductID] == nil {
				stockUpdates[productItem.Product.ProductID] = make(map[string][]int)
			}

			for color, quantities := range productItem.Product.Color {
				colorKey := strings.ToLower(color)
				if existing, exists := stockUpdates[productItem.Product.ProductID][colorKey]; exists {
					for i, qty := range quantities {
						if i < len(existing) {
							stockUpdates[productItem.Product.ProductID][colorKey][i] += qty
						}
					}

				} else {
					//initialize with current quantities
					stockUpdates[productItem.Product.ProductID][colorKey] = make([]int, len(quantities))
					copy(stockUpdates[productItem.Product.ProductID][colorKey], quantities)
				}

			}
		}
	}
	return stockUpdates, purchaseEntries, nil
}
