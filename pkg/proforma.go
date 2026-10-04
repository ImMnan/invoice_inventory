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
		for _, prod := range productItem.Product {
			stockKey := StockKey(prod.ProductID, prod.Print)
			if productItem.Type == "proforma" && productItem.Type != "purchase-invoice" && productItem.Type != "job" {
				saleEntry := Proforma{
					UUID:     uuid.New().String(),
					Type:     "sale",
					For:      productItem.For,
					Invoice:  productItem.Invoice,
					Date:     productItem.Date,
					IsPaid:   false,
					Rejected: false,
					Product: []ProductStruct{{
						ProductID: prod.ProductID,
						Print:     NormalizePrint(prod.Print),
						Gen:       prod.Gen,
						GST:       prod.GST,
						Color:     make(map[string][]int),
						Quantity:  prod.Quantity,
						Total:     prod.Total,
					}},
				}
				for color, quantities := range prod.Color {
					saleEntry.Product[0].Color[strings.ToLower(strings.TrimSpace(color))] = quantities
				}
				saleEntries = append(saleEntries, saleEntry)
				if stockUpdates[stockKey] == nil {
					stockUpdates[stockKey] = make(map[string][]int)
				}
				for color, quantities := range prod.Color {
					colorKey := strings.ToLower(strings.TrimSpace(color))
					if existing, exists := stockUpdates[stockKey][colorKey]; exists {
						for i, qty := range quantities {
							if i < len(existing) {
								existing[i] += qty
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
	return stockUpdates, saleEntries, nil
}
