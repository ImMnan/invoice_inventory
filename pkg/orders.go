// Package pkg provides inventory management functionality for processing
// proforma invoices and purchase orders, updating stock levels, and
// maintaining transaction history in JSON format.

package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func (stkUp *StockUpdate) dataCalculation(currentStock map[string]map[string][]int) error {
	if stkUp.proformaStkUpdates == nil && stkUp.purchaseStkUpdates == nil {
		return fmt.Errorf("stock updates are not initialized")
	}

	if stkUp.proformaStkUpdates != nil {
		for productUID, colors := range stkUp.proformaStkUpdates {
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

		if stkUp.purchaseStkUpdates != nil {
			for productUID, colors := range stkUp.purchaseStkUpdates {
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

// ProcessInventoryUpdate - Main function to process proforma and purchase data and update inventory
//func (data *JsLocalDB) ProcessInventoryUpdate(productData *ProductSlice) error {
//	invoiceData, err := data.ProcessInventoryUpdateWithInvoiceData(productData)
//	if err != nil {
//		return err
//	}
//	_ = invoiceData // Suppress unused variable warning
//	return nil
//}

// ProcessInventoryUpdateWithInvoiceData - Enhanced version that returns invoice-grouped data
func (data *JsLocalDB) ProcessInventoryUpdateWithInvoiceData(productData *ProductSlice) (*InvoiceGroupedData, error) {
	// Step 1: Process proforma items and get stock updates + sale entries
	proformaStockUpdates, saleEntries, err := productData.addProforma()
	if err != nil {
		return nil, fmt.Errorf("failed to process proforma data: %w", err)
	}

	// Step 2: Process purchase items and get stock updates + purchase entries
	purchaseStockUpdates, purchaseEntries, err := productData.addPurchase()
	if err != nil {
		return nil, fmt.Errorf("failed to process purchase data: %w", err)
	}

	// Step 3: Create enhanced StockUpdate struct with all data
	stockUpdate := &StockUpdate{
		proformaStkUpdates: proformaStockUpdates,
		purchaseStkUpdates: purchaseStockUpdates,
		saleEntries:        saleEntries,
		purchaseEntries:    purchaseEntries,
	}

	// Step 4: Update inventory with everything in the stockUpdate struct
	invoiceGroupedData, err := data.updateInventoryFromStockUpdate(stockUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Return the invoice-grouped data for invoice generation
	return invoiceGroupedData, nil
}

// updateInventoryFromStockUpdate - Simplified version that gets all data from StockUpdate struct
func (data *JsLocalDB) updateInventoryFromStockUpdate(stockUpdate *StockUpdate) (*InvoiceGroupedData, error) {
	// Step 1: Read current inventory
	inventory, err := os.Open(data.file)
	if err != nil {
		return nil, fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer inventory.Close()

	var allEntries []In_stockTshirtStruct
	if err := json.NewDecoder(inventory).Decode(&allEntries); err != nil {
		return nil, fmt.Errorf("failed to decode inventory: %w", err)
	}

	// Step 2: Get current stock levels using existing function
	currentStock, err := data.getExistingStock()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing stock: %w", err)
	}

	// Step 3: Apply stock calculations using existing dataCalculation function
	if err := stockUpdate.dataCalculation(currentStock); err != nil {
		return nil, fmt.Errorf("failed to calculate stock updates: %w", err)
	}

	// Step 4: Create grouped data for invoice generation
	invoiceGroupedData := &InvoiceGroupedData{
		SalesByInvoice:        make(map[string][]Proforma),
		StockChangesByInvoice: make(map[string]map[string]map[string][]int),
	}

	// Group sale entries by invoice ID
	for _, saleEntry := range stockUpdate.saleEntries {
		invoiceID := saleEntry.Invoice
		invoiceGroupedData.SalesByInvoice[invoiceID] = append(invoiceGroupedData.SalesByInvoice[invoiceID], saleEntry)

		// Track stock changes for this invoice
		if invoiceGroupedData.StockChangesByInvoice[invoiceID] == nil {
			invoiceGroupedData.StockChangesByInvoice[invoiceID] = make(map[string]map[string][]int)
		}

		productUID := saleEntry.Product.UID
		if invoiceGroupedData.StockChangesByInvoice[invoiceID][productUID] == nil {
			invoiceGroupedData.StockChangesByInvoice[invoiceID][productUID] = make(map[string][]int)
		}

		// Add quantities for this sale entry
		for color, quantities := range saleEntry.Product.Color {
			colorKey := strings.ToLower(color)
			if existing, exists := invoiceGroupedData.StockChangesByInvoice[invoiceID][productUID][colorKey]; exists {
				// Add to existing quantities
				for i, qty := range quantities {
					if i < len(existing) {
						existing[i] += qty
					}
				}
			} else {
				// Initialize with current quantities
				invoiceGroupedData.StockChangesByInvoice[invoiceID][productUID][colorKey] = make([]int, len(quantities))
				copy(invoiceGroupedData.StockChangesByInvoice[invoiceID][productUID][colorKey], quantities)
			}
		}
	}

	// Note: Purchase entries are still added to inventory for record keeping,
	// but they don't need to be grouped by invoice for invoice generation
	// since purchases don't generate customer invoices

	// Step 5: Update in_stock entries with calculated values
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
		return nil, fmt.Errorf("failed to marshal updated inventory: %w", err)
	}

	if err := os.WriteFile(data.file, updatedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write updated inventory: %w", err)
	}

	return invoiceGroupedData, nil
}

// ExampleUsage demonstrates the enhanced integration
func ExampleUsage() error {
	// Step 1: Initialize database connection with inventory file path
	db := &JsLocalDB{file: "Data/inventory.json"}

	// Step 2: Load your product data from CSV or manual input
	var productData ProductSlice
	// ... populate productData with proforma and purchase items ...

	// Step 3: Process all updates and get invoice-grouped data
	invoiceGroupedData, err := db.ProcessInventoryUpdateWithInvoiceData(&productData)
	if err != nil {
		return fmt.Errorf("inventory update failed: %w", err)
	}

	// Step 4: Generate invoices using the simplified enhanced function
	err = productData.makeInvoiceWithStockData("table", invoiceGroupedData)
	if err != nil {
		return fmt.Errorf("failed to generate invoices: %w", err)
	}

	fmt.Println("✅ Inventory updated and invoices generated successfully!")
	return nil
}
