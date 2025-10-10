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
						Print:     prod.Print,
						Gen:       prod.Gen,
						GST:       prod.GST,
						Color:     make(map[string][]int),
						Quantity:  prod.Quantity,
						Total:     prod.Total,
					}},
				}
				for color, quantities := range prod.Color {
					saleEntry.Product[0].Color[strings.ToLower(color)] = quantities
				}
				saleEntries = append(saleEntries, saleEntry)
				if stockUpdates[prod.ProductID] == nil {
					stockUpdates[prod.ProductID] = make(map[string][]int)
				}
				for color, quantities := range prod.Color {
					colorKey := strings.ToLower(color)
					if existing, exists := stockUpdates[prod.ProductID][colorKey]; exists {
						for i, qty := range quantities {
							if i < len(existing) {
								existing[i] += qty
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
	return stockUpdates, saleEntries, nil
}
