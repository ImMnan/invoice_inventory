// Package pkg provides inventory management functionality for processing 
// proforma invoices and purchase orders, updating stock levels, and 
// maintaining transaction history in JSON format.

package pkg

import (
	"encoding/json"
	"fmt"
	"os"
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

func (product *ProductSlice) addPurchase() (map[string]map[string][]int, []Purchase, error) {

	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	var purchaseEntries []Purchase

	for _, productItem := range *product {
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

func (p *StockUpdate) dataCalculation(currentStock map[string]map[string][]int) error {
	if p.proformaStkUpdates == nil && p.purchaseStkUpdates == nil {
		return fmt.Errorf("stock updates are not initialized")
	}

	if p.proformaStkUpdates != nil {
		for productUID, colors := range p.proformaStkUpdates {
			if currentStock[productUID] != nil {
				for color, subtractQuantities := range colors {
					// Try to find matching color (case-insensitive)
					var matchingStockColor string
					var stockQuantities []int
					found := false

					for stockColor, quantities := range currentStock[productUID] {
						if strings.EqualFold(stockColor, color) {
							matchingStockColor = stockColor
							stockQuantities = quantities
							found = true
							break
						}
					}

					if found {
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
						currentStock[productUID][matchingStockColor] = stockQuantities
					} else {
						return fmt.Errorf("\nerror: color '%s' not found in existing stock for product %s", color, productUID)
					}
				}
			} else {
				// New product - check if we're trying to subtract from non-existent stock
				return fmt.Errorf("\nerror: product '%s' not found in existing stock", productUID)
			}

		}
		// Process for purchase stock updates

		if p.purchaseStkUpdates != nil {
			for productUID, colors := range p.purchaseStkUpdates {
				if currentStock[productUID] != nil {
					for color, addQuantities := range colors {
						// Try to find matching color (case-insensitive)
						var matchingStockColor string
						var stockQuantities []int
						found := false

						for stockColor, quantities := range currentStock[productUID] {
							if strings.EqualFold(stockColor, color) {
								matchingStockColor = stockColor
								stockQuantities = quantities
								found = true
								break
							}
						}
						if found {
							for i, addQty := range addQuantities {
								if i < len(stockQuantities) {
									stockQuantities[i] += addQty
									// Uncomment for debugging
									//fmt.Printf("Adding %d to %s %s size %d: %d -> %d\n", addQty, productUID, color, i, stockQuantities[i]-addQty, stockQuantities[i])
								} else {
									return fmt.Errorf("\nerror: size index %d out of range for product %s color %s", i, productUID, color)
								}
							}
						}
						currentStock[productUID][matchingStockColor] = stockQuantities
						// create a new color entry if it doesn't exist
						if !found {
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
	}
	return nil
}

// now, I want the in_stock values to be updated in the inventory json file, based on the above stock updates. 
// We need to add the proforma or purhchase into the json as well as update the in_stock values.
// in_stock values can be taken from the data *JsLocalDB struct, which has the file path to the inventory json file.
// Now, the updateStock function will read the existing stock from the inventory file, apply the stock updates, and write the updated stock back to the file.
// We can also use dataCalculation() function as it has the calculation logic for stock updates.


// ProcessInventoryUpdate - Main function to process proforma and purchase data and update inventory
func (data *JsLocalDB) ProcessInventoryUpdate(productData *ProductSlice) error {
	// Step 1: Process proforma items and get stock updates + sale entries
	proformaStockUpdates, saleEntries, err := productData.addProforma()
	if err != nil {
		return fmt.Errorf("failed to process proforma data: %w", err)
	}

	// Step 2: Process purchase items and get stock updates + purchase entries
	purchaseStockUpdates, purchaseEntries, err := productData.addPurchase()
	if err != nil {
		return fmt.Errorf("failed to process purchase data: %w", err)
	}

	// Step 3: Create enhanced StockUpdate struct with all data
	stockUpdate := &StockUpdate{
		proformaStkUpdates: proformaStockUpdates,
		purchaseStkUpdates: purchaseStockUpdates,
		saleEntries:        saleEntries,
		purchaseEntries:    purchaseEntries,
	}

	// Step 4: Update inventory with everything in the stockUpdate struct
	if err := data.updateInventoryFromStockUpdate(stockUpdate); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

// updateInventoryFromStockUpdate - Simplified version that gets all data from StockUpdate struct
func (data *JsLocalDB) updateInventoryFromStockUpdate(stockUpdate *StockUpdate) error {
	// Step 1: Read current inventory
	inventory, err := os.Open(data.file)
	if err != nil {
		return fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer inventory.Close()

	var allEntries []In_stockTshirtStruct
	if err := json.NewDecoder(inventory).Decode(&allEntries); err != nil {
		return fmt.Errorf("failed to decode inventory: %w", err)
	}

	// Step 2: Get current stock levels using existing function
	currentStock, err := data.existingStock()
	if err != nil {
		return fmt.Errorf("failed to get existing stock: %w", err)
	}

	// Step 3: Apply stock calculations using existing dataCalculation function
	if err := stockUpdate.dataCalculation(currentStock); err != nil {
		return fmt.Errorf("failed to calculate stock updates: %w", err)
	}

	// Step 4: Update in_stock entries with calculated values
	for i := range allEntries {
		if allEntries[i].Type == "in_stock" {
			productUID := allEntries[i].Product.UID
			if updatedStock, exists := currentStock[productUID]; exists {
				// Update the color quantities with calculated values
				for color, newQuantities := range updatedStock {
					// Find matching color in existing entry (case-insensitive)
					for existingColor := range allEntries[i].Product.Color {
						if strings.EqualFold(existingColor, color) {
							allEntries[i].Product.Color[existingColor] = make([]int, len(newQuantities))
							copy(allEntries[i].Product.Color[existingColor], newQuantities)
							break
						}
					}
				}
				// Recalculate total for this product
				total := 0
				for _, quantities := range allEntries[i].Product.Color {
					for _, qty := range quantities {
						total += qty
					}
				}
				allEntries[i].Product.Total = total
			}
		}
	}

	// Step 5: Add new sale entries from stockUpdate
	for _, saleEntry := range stockUpdate.saleEntries {
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

	// Step 6: Add new purchase entries from stockUpdate
	for _, purchaseEntry := range stockUpdate.purchaseEntries {
		newEntry := In_stockTshirtStruct{
			UUID:     purchaseEntry.UUID,
			Type:     purchaseEntry.Type,
			From:     purchaseEntry.From,
			Invoice:  purchaseEntry.Invoice,
			Date:     purchaseEntry.Date,
			IsPaid:   false, // Default for purchases
			Rejected: false, // Default for purchases
			Product:  purchaseEntry.Product,
		}
		allEntries = append(allEntries, newEntry)
	}

	// Step 7: Write updated inventory back to file
	updatedData, err := json.MarshalIndent(allEntries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	if err := os.WriteFile(data.file, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated inventory: %w", err)
	}

	return nil
}

// updateInventory - Complete rewrite to properly update inventory with stock calculations and new entries
func (data *JsLocalDB) updateInventory(stockUpdate *StockUpdate, saleEntries []Proforma, purchaseEntries []Purchase) error {
	// Step 1: Read current inventory
	inventory, err := os.Open(data.file)
	if err != nil {
		return fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer inventory.Close()

	var allEntries []In_stockTshirtStruct
	if err := json.NewDecoder(inventory).Decode(&allEntries); err != nil {
		return fmt.Errorf("failed to decode inventory: %w", err)
	}

	// Step 2: Get current stock levels using existing function
	currentStock, err := data.existingStock()
	if err != nil {
		return fmt.Errorf("failed to get existing stock: %w", err)
	}

	// Step 3: Apply stock calculations using existing dataCalculation function
	if err := stockUpdate.dataCalculation(currentStock); err != nil {
		return fmt.Errorf("failed to calculate stock updates: %w", err)
	}

	// Step 4: Update in_stock entries with calculated values
	for i := range allEntries {
		if allEntries[i].Type == "in_stock" {
			productUID := allEntries[i].Product.UID
			if updatedStock, exists := currentStock[productUID]; exists {
				// Update the color quantities with calculated values
				for color, newQuantities := range updatedStock {
					// Find matching color in existing entry (case-insensitive)
					for existingColor := range allEntries[i].Product.Color {
						if strings.EqualFold(existingColor, color) {
							allEntries[i].Product.Color[existingColor] = make([]int, len(newQuantities))
							copy(allEntries[i].Product.Color[existingColor], newQuantities)
							break
						}
					}
				}
				// Recalculate total for this product
				total := 0
				for _, quantities := range allEntries[i].Product.Color {
					for _, qty := range quantities {
						total += qty
					}
				}
				allEntries[i].Product.Total = total
			}
		}
	}

	// Step 5: Add new sale entries to inventory
	for _, saleEntry := range saleEntries {
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

	// Step 6: Add new purchase entries to inventory
	for _, purchaseEntry := range purchaseEntries {
		newEntry := In_stockTshirtStruct{
			UUID:     purchaseEntry.UUID,
			Type:     purchaseEntry.Type,
			From:     purchaseEntry.From,
			Invoice:  purchaseEntry.Invoice,
			Date:     purchaseEntry.Date,
			IsPaid:   false, // Default for purchases
			Rejected: false, // Default for purchases
			Product:  purchaseEntry.Product,
		}
		allEntries = append(allEntries, newEntry)
	}

	// Step 7: Write updated inventory back to file
	updatedData, err := json.MarshalIndent(allEntries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	if err := os.WriteFile(data.file, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated inventory: %w", err)
	}

	return nil
}

// Example usage function showing how to use the new simplified inventory update system
func ExampleInventoryUpdate() error {
	// Step 1: Initialize database connection with inventory file path
	db := &JsLocalDB{file: "Data/inventory.json"}
	
	// Step 2: Load your product data from CSV or manual input
	// This would typically come from ProcessProductData() or manual data entry
	var productData ProductSlice
	// ... populate productData with proforma and purchase items ...
	
	// Step 3: Process all updates in a single transaction - everything from StockUpdate!
	if err := db.ProcessInventoryUpdate(&productData); err != nil {
		return fmt.Errorf("inventory update failed: %w", err)
	}
	
	fmt.Println("Inventory updated successfully!")
	return nil
}

// Legacy updateStock function - keeping for backward compatibility
func (data *JsLocalDB) updateStock(stockUpdate *StockUpdate) error {
	inventory, err := os.Open(data.file)
	if err != nil {
		return err
	}
	defer inventory.Close()
	var stock []In_stockTshirtStruct
	if err := json.NewDecoder(inventory).Decode(&stock); err != nil {
		return err
	}

	for _, item := range stock {
		if item.Type == "in_stock" {
			// Update stock quantities based on the stockUpdate
			for productUID, colorUpdates := range stockUpdate.proformaStkUpdates {
				for color, quantities := range colorUpdates {
					// Find the product in the existing stock
					for i := range stock {
						if stock[i].Product.UID == productUID {
							// Update the color quantities
							stock[i].Product.Color[color] = quantities
							break
						}
					}
				}
			}

			for productUID, colorUpdates := range stockUpdate.purchaseStkUpdates {
				for color, quantities := range colorUpdates {
					// Find the product in the existing stock
					for i := range stock {
						if stock[i].Product.UID == productUID {
							// Update the color quantities
							stock[i].Product.Color[color] = quantities
							break
						}
					}
				}
			}

			// Write the updated stock back to the inventory file
			updatedData, err := json.MarshalIndent(stock, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal updated inventory: %v", err)
			}

			if err := os.WriteFile(data.file, updatedData, 0644); err != nil {
				return fmt.Errorf("failed to write updated inventory: %v", err)
			}
		}
	}
	return nil
}
