package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
										productUID, color, i, stockQuantities[i], subtractQty)
								}
							}
						}
						// No need to reassign since we modified the original slice
					} else {
						return fmt.Errorf("\nerror: color '%s' not found in existing stock for product %s", color, productUID)
					}
				}
			} else {
				// New product - check if we're trying to subtract from non-existent stock
				return fmt.Errorf("\nerror: product '%s' not found in existing stock", productUID)
			}
		}
	}

	// Process purchase stock updates (add quantities)
	if stkUp.purchaseStkUpdates != nil {
		for productUID, colors := range stkUp.purchaseStkUpdates {
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
								return fmt.Errorf("\nerror: size index %d out of range for product %s color %s", i, productUID, color)
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
				return fmt.Errorf("\nerror: product '%s' not found in existing stock", productUID)
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

	// Step 3: Update in_stock entries with calculated values
	for i := range allEntries {
		if allEntries[i].Type == "in_stock" {
			productUID := allEntries[i].Product.ProductID
			if updatedStock, exists := currentStock[productUID]; exists {
				allEntries[i].Product.Color = make(map[string][]int)
				for color, newQuantities := range updatedStock {
					allEntries[i].Product.Color[color] = make([]int, len(newQuantities))
					copy(allEntries[i].Product.Color[color], newQuantities)
				}
				quantity := 0
				for _, quantities := range allEntries[i].Product.Color {
					for _, qty := range quantities {
						quantity += qty
					}
				}
				allEntries[i].Product.Quantity = quantity
			}
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

	// Step 6: Write updated inventory back to file
	updatedData, err := json.MarshalIndent(allEntries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	if err := os.WriteFile(data.InventoryFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated inventory: %w", err)
	}

	return nil
}
