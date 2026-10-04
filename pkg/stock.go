package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
)

func (data *JsLocalDB) Stocks() ([]byte, error) {
	// open and read json
	var stock []TshirtStruct
	file, err := os.Open(data.InventoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fileData, &stock)
	if err != nil {
		return nil, err
	}

	// Return the entire stock data as JSON
	stockData, err := json.Marshal(stock)
	if err != nil {
		return nil, err
	}
	return stockData, nil
}

func (stkUp *StockUpdate) dataCalculation(currentStock map[string]map[string][]int) error {

	if stkUp.proformaStkUpdates == nil && stkUp.purchaseStkUpdates == nil {
		return fmt.Errorf("stock updates are not initialized")
	}
	// Process proforma stock updates (subtract quantities)
	if stkUp.proformaStkUpdates != nil {
		for productUID, colors := range stkUp.proformaStkUpdates {
			productLabel := strings.ReplaceAll(productUID, "\x00", " design ")
			if currentStock[productUID] != nil {
				for color, subtractQuantities := range colors {
					// Try to find matching color (case-insensitive)
					var matchingStockColor string
					found := false

					for stockColor := range currentStock[productUID] {
						if strings.EqualFold(stockColor, color) {
							matchingStockColor = stockColor
							found = true
							break
						}
					}

					if found {
						// Directly modify the slice in currentStock
						stockQuantities := currentStock[productUID][matchingStockColor]
						for i, subtractQty := range subtractQuantities {
							if i < len(stockQuantities) {
								if stockQuantities[i] >= subtractQty {
									stockQuantities[i] -= subtractQty
									// Uncomment for debugging
									//fmt.Printf("Subtracting %d from %s %s size %d: %d -> %d\n", subtractQty, productUID, color, i, stockQuantities[i]+subtractQty, stockQuantities[i])
								} else {
									return fmt.Errorf("\nerror: not enough stock for %s %s size %d. available: %d, requested: %d",
										productLabel, color, i, stockQuantities[i], subtractQty)
								}
							}
						}
						// No need to reassign since we modified the original slice
					} else {
						return fmt.Errorf("\nerror: color '%s' not found in existing stock for product %s", color, productLabel)
					}
				}
			} else {
				// New product - check if we're trying to subtract from non-existent stock
				return fmt.Errorf("\nerror: product '%s' not found in existing stock", productLabel)
			}
		}
	}

	// Process purchase stock updates (add quantities)
	if stkUp.purchaseStkUpdates != nil {
		for productUID, colors := range stkUp.purchaseStkUpdates {
			productLabel := strings.ReplaceAll(productUID, "\x00", " design ")
			if currentStock[productUID] == nil {
				currentStock[productUID] = make(map[string][]int)
			}
			if currentStock[productUID] != nil {
				for color, addQuantities := range colors {
					// Try to find matching color (case-insensitive)
					var matchingStockColor string
					found := false

					for stockColor := range currentStock[productUID] {
						if strings.EqualFold(stockColor, color) {
							matchingStockColor = stockColor
							found = true
							break
						}
					}
					if found {
						// Directly modify the slice in currentStock
						stockQuantities := currentStock[productUID][matchingStockColor]
						for i, addQty := range addQuantities {
							if i < len(stockQuantities) {
								stockQuantities[i] += addQty
								// Uncomment for debugging
								//fmt.Printf("Adding %d to %s %s size %d: %d -> %d\n", addQty, productUID, color, i, stockQuantities[i]-addQty, stockQuantities[i])
							} else {
								return fmt.Errorf("\nerror: size index %d out of range for product %s color %s", i, productLabel, color)
							}
						}
						// No need to reassign since we modified the original slice
					} else {
						// create a new color entry if it doesn't exist
						currentStock[productUID][color] = make([]int, len(addQuantities))
						copy(currentStock[productUID][color], addQuantities)
						// Uncomment for debugging
						//fmt.Printf("Adding new color %s for product %s with quantities: %v\n", color, productUID, addQuantities)
					}
				}
			} else {
				// New product - check if we're trying to add to non-existent stock
				return fmt.Errorf("\nerror: product '%s' not found in existing stock", productLabel)
			}
		}
	}
	return nil
}

func (data *JsLocalDB) UpdateInventoryFromStockUpdate(stockUpdate *StockUpdate) error {
	// Step 1: Get current inventory and in_stock values
	allEntries, currentStock, err := data.getExistingStock()
	if err != nil {
		return fmt.Errorf("failed to get existing stock: %w", err)
	}

	// Step 2: Apply stock calculations (subtract for sales, add for purchases)
	if err := stockUpdate.dataCalculation(currentStock); err != nil {
		return fmt.Errorf("failed to calculate stock updates: %w", err)
	}

	// Step 3: Update in_stock entries with calculated values (handle Product as a slice)
	seen := make(map[string]bool)
	stockEntryByProduct := make(map[string]int)
	productMetadata := make(map[string]ProductStruct)
	stockEntryIndex := -1
	for i := range allEntries {
		if allEntries[i].Type == "in_stock" {
			if stockEntryIndex == -1 {
				stockEntryIndex = i
			}
			var products []ProductStruct
			for j := range allEntries[i].Product {
				productID := allEntries[i].Product[j].ProductID
				if _, exists := stockEntryByProduct[productID]; !exists {
					stockEntryByProduct[productID] = i
					productMetadata[productID] = allEntries[i].Product[j]
				}
				allEntries[i].Product[j].Print = NormalizePrint(allEntries[i].Product[j].Print)
				productUID := StockKey(allEntries[i].Product[j].ProductID, allEntries[i].Product[j].Print)
				if seen[productUID] {
					continue
				}
				seen[productUID] = true
				if updatedStock, exists := currentStock[productUID]; exists {
					allEntries[i].Product[j].Color = make(map[string][]int)
					for color, newQuantities := range updatedStock {
						allEntries[i].Product[j].Color[color] = make([]int, len(newQuantities))
						copy(allEntries[i].Product[j].Color[color], newQuantities)
					}
					quantity := 0
					for _, quantities := range allEntries[i].Product[j].Color {
						for _, qty := range quantities {
							quantity += qty
						}
					}
					allEntries[i].Product[j].Quantity = quantity
				}
				products = append(products, allEntries[i].Product[j])
			}
			allEntries[i].Product = products
		}
	}
	catalogLoaded := false
	for _, purchase := range stockUpdate.PurchaseEntries {
		for _, product := range purchase.Product {
			key := StockKey(product.ProductID, product.Print)
			if seen[key] {
				continue
			}
			if !catalogLoaded && data.ProductFile != "" {
				catalog, err := data.getProductData()
				if err != nil {
					return fmt.Errorf("failed to load product metadata: %w", err)
				}
				for _, catalogProduct := range catalog {
					metadata := productMetadata[catalogProduct.ProductID]
					metadata.ProductID = catalogProduct.ProductID
					if catalogProduct.Name != "" {
						metadata.Name = catalogProduct.Name
					}
					if catalogProduct.Description != "" {
						metadata.Description = catalogProduct.Description
					}
					if catalogProduct.Gen != "" {
						metadata.Gen = catalogProduct.Gen
					}
					if catalogProduct.GST != 0 {
						metadata.GST = catalogProduct.GST
					}
					if catalogProduct.Price != 0 {
						metadata.Price = catalogProduct.Price
					}
					productMetadata[catalogProduct.ProductID] = metadata
				}
				catalogLoaded = true
			}
			seen[key] = true
			if metadata, exists := productMetadata[product.ProductID]; exists {
				product.Name = metadata.Name
				product.Description = metadata.Description
				product.GST = metadata.GST
				product.Price = metadata.Price
				if metadata.Gen != "" {
					product.Gen = metadata.Gen
				}
			}
			product.Print = NormalizePrint(product.Print)
			product.Color = currentStock[key]
			product.Quantity = 0
			product.Total = 0
			for _, quantities := range product.Color {
				for _, quantity := range quantities {
					product.Quantity += quantity
				}
			}
			if stockEntryIndex == -1 {
				stockEntryIndex = len(allEntries)
				allEntries = append(allEntries, In_stockTshirtStruct{
					UUID: uuid.New().String(), Type: "in_stock", Invoice: "NA",
				})
			}
			targetIndex, exists := stockEntryByProduct[product.ProductID]
			if !exists {
				targetIndex = stockEntryIndex
				stockEntryByProduct[product.ProductID] = targetIndex
			}
			allEntries[targetIndex].Product = append(allEntries[targetIndex].Product, product)
		}
	}

	// Step 4: Add new sale entries from stockUpdate (transaction history)
	for _, saleEntry := range stockUpdate.SaleEntries {
		newEntry := In_stockTshirtStruct{
			UUID:     saleEntry.UUID,
			Type:     saleEntry.Type,
			For:      saleEntry.For,
			Invoice:  saleEntry.Invoice,
			Date:     saleEntry.Date,
			IsPaid:   saleEntry.IsPaid,
			Rejected: saleEntry.Rejected,
			Product:  saleEntry.Product,
		}
		allEntries = append(allEntries, newEntry)
	}

	// Step 5: Add new purchase entries from stockUpdate (transaction history)
	for _, purchaseEntry := range stockUpdate.PurchaseEntries {
		newEntry := In_stockTshirtStruct{
			UUID:     purchaseEntry.UUID,
			Type:     purchaseEntry.Type,
			From:     purchaseEntry.From,
			Invoice:  purchaseEntry.Invoice,
			Date:     purchaseEntry.Date,
			IsPaid:   false,
			Rejected: false,
			Product:  purchaseEntry.Product,
		}
		allEntries = append(allEntries, newEntry)
	}

	// Step 6: Group entries by invoice and write updated inventory back to file
	grouped := make(map[string]*In_stockTshirtStruct)

	for _, entry := range allEntries {
		if entry.Type == "in_stock" && len(entry.Product) == 0 {
			continue
		}
		key := entry.Invoice + "|" + entry.Type + "|" + entry.Date
		if group, ok := grouped[key]; ok {
			group.Product = append(group.Product, entry.Product...)
		} else {
			// Copy entry, but ensure Product is a slice
			newEntry := entry
			newEntry.Product = append([]ProductStruct{}, entry.Product...)
			grouped[key] = &newEntry
		}
	}

	var groupedEntries []In_stockTshirtStruct
	for _, v := range grouped {
		groupedEntries = append(groupedEntries, *v)
	}

	updatedData, err := json.MarshalIndent(groupedEntries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	if err := os.WriteFile(data.InventoryFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated inventory: %w", err)
	}

	return nil
}

// GetExistingStock provides public access to existing stock data
func (data *JsLocalDB) GetExistingStock() ([]In_stockTshirtStruct, map[string]map[string][]int, error) {
	return data.getExistingStock()
}
