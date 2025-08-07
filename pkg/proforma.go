package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func (p ProBool) ProcessFileData(data [][]string) ([]byte, error) {
	//	var proformaData []TshirtStruct
	if p.active {
		var proformaData []TshirtStruct
		poData := data
		for i, row := range poData {
			if row[0] != "proforma" { // Ensure we only process proforma rows
				return nil, fmt.Errorf("invalid proforma data: expected 'proforma' type in first column")
			}
			if i == 0 { // Skip header row
				continue
			}
			if len(row) < 18 { // Ensure we have all required columns
				continue
			}

			// Parse size quantities
			colorMap := make(map[string][]int)
			quantities := make([]int, 8)

			// Parse quantities for each size (columns 8-15)
			for j := 0; j < 8; j++ {
				if qty, err := strconv.Atoi(strings.TrimSpace(row[10+j])); err == nil {
					quantities[j] = qty
				}
			}

			// Color is in column 9, use it as key for the map
			color := strings.TrimSpace(row[9])
			colorMap[color] = quantities

			// Parse total (column 16)
			total := 0
			if totalStr := strings.TrimSpace(row[18]); totalStr != "" {
				if t, err := strconv.Atoi(totalStr); err == nil {
					total = t
				}
			}

			proformaData = append(proformaData, Proforma{
				UUID:     uuid.New().String(),
				Type:     strings.TrimSpace(row[0]),
				Invoice:  strings.TrimSpace(row[1]),
				For:      strings.TrimSpace(row[2]), // For field
				Date:     strings.TrimSpace(row[4]),
				IsPaid:   false, // Default to false for proforma
				Rejected: false, // Default to false for proforma
				Product: ProductStruct{
					UID:   strings.TrimSpace(row[3]), // Product_Id
					Print: strings.TrimSpace(row[7]), // Print
					Gen:   strings.TrimSpace(row[8]), // Gen
					Price: func() int {
						priceInt, err := strconv.Atoi(strings.TrimSpace(row[5]))
						if err != nil {
							return 0
						}
						return priceInt
					}(),
					GST:   strings.TrimSpace(row[6]), // GST as string
					Color: colorMap,
					Total: total,
				},
			})
		}
		return json.Marshal(proformaData)

	}
	return nil, nil
}

func (p *ProBool) ApplyFileData(data [][]string, format string) error {
	proformaData, err := p.ProcessFileData(data)
	if err != nil {
		return fmt.Errorf("\nerror processing proforma data:\n %v", err)
	}

	// Parse proforma data to extract customer IDs and group by customer
	var proformaItems []Proforma
	if err := json.Unmarshal(proformaData, &proformaItems); err != nil {
		return fmt.Errorf("failed to parse proforma data: %v", err)
	}

	// Group proforma items by invoice ID
	invoiceGroups := make(map[string][]Proforma)
	for _, item := range proformaItems {
		invoiceID := item.Invoice
		invoiceGroups[invoiceID] = append(invoiceGroups[invoiceID], item)
	}

	// Process inventory updates and generate invoices for each invoice group
	for invoiceID, invoiceItems := range invoiceGroups {
		// Extract customer ID from the first item in this invoice
		var customerID string
		if len(invoiceItems) > 0 {
			customerID = invoiceItems[0].For
		}

		if customerID == "" {
			fmt.Printf("Warning: No customer ID found for invoice %s, skipping...\n", invoiceID)
			continue
		}

		// Process inventory updates for this invoice
		if err := addProforma(invoiceItems); err != nil {
			return fmt.Errorf("\nerror adding proforma for invoice %s:\n %v", invoiceID, err)
		}
	}

	// Generate all invoices at once
	if err := makeInvoice(proformaData, format, invoiceGroups); err != nil {
		return fmt.Errorf("\nerror creating invoices:\n %v", err)
	}

	if len(invoiceGroups) == 0 {
		return fmt.Errorf("no invoice IDs found in proforma data")
	}
	//	fmt.Printf("\nProcessed and generated %d invoices\n", len(invoiceGroups))
	return nil

}

func addProforma(proformaItems []Proforma) error {
	inventory, err := os.Open("Data/inventory.json")
	if err != nil {
		return err
	}
	defer inventory.Close()
	var stock []Proforma

	if err := json.NewDecoder(inventory).Decode(&stock); err != nil {
		return err
	}

	// Extract customer ID from the first proforma item
	//var customerID string
	//if len(proformaItems) > 0 {
	//	customerID = proformaItems[0].For
	//}

	// Create sale entries and track stock changes
	var saleEntries []Proforma
	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities

	for _, proformaItem := range proformaItems {
		// Create a sale entry from the proforma item
		saleEntry := Proforma{
			UUID:     uuid.New().String(),
			Type:     "sale",
			For:      proformaItem.For, // Copy customer ID
			Invoice:  proformaItem.Invoice,
			Date:     proformaItem.Date,
			IsPaid:   false,
			Rejected: false,
			Product: ProductStruct{
				UID:   proformaItem.Product.UID,
				Print: proformaItem.Product.Print,
				Gen:   proformaItem.Product.Gen,
				GST:   proformaItem.Product.GST,
				Color: make(map[string][]int),
				Total: proformaItem.Product.Total,
			},
		}

		// Copy color data to sale entry (normalize color keys to lowercase)
		for color, quantities := range proformaItem.Product.Color {
			saleEntry.Product.Color[strings.ToLower(color)] = quantities
		}

		saleEntries = append(saleEntries, saleEntry)

		// Initialize stock updates for this product if not exists
		if stockUpdates[proformaItem.Product.UID] == nil {
			stockUpdates[proformaItem.Product.UID] = make(map[string][]int)
		}

		// Track quantities to subtract from stock
		for color, quantities := range proformaItem.Product.Color {
			colorKey := strings.ToLower(color)
			if existing, exists := stockUpdates[proformaItem.Product.UID][colorKey]; exists {
				// Add to existing quantities
				for i, qty := range quantities {
					if i < len(existing) {
						existing[i] += qty
					}
				}
			} else {
				// Initialize with current quantities
				stockUpdates[proformaItem.Product.UID][colorKey] = make([]int, len(quantities))
				copy(stockUpdates[proformaItem.Product.UID][colorKey], quantities)
			}
		}
	}

	// Collect current stock state and apply subtractions
	currentStock := make(map[string]map[string][]int) // productUID -> color -> quantities

	// First, collect all existing in_stock data
	for _, item := range stock {
		if item.Type == "in_stock" {
			if currentStock[item.Product.UID] == nil {
				currentStock[item.Product.UID] = make(map[string][]int)
			}
			for color, quantities := range item.Product.Color {
				currentStock[item.Product.UID][color] = make([]int, len(quantities))
				copy(currentStock[item.Product.UID][color], quantities)
			}
		}
	}

	// Apply subtractions to the inventory
	if err := proformaCal(stockUpdates, currentStock); err != nil {
		return fmt.Errorf("error during proforma calculation: %v", err)
	}

	// Remove all existing in_stock entries and keep other entries
	if err := saleConsolidation(stock, saleEntries, currentStock); err != nil {
		return fmt.Errorf("error during consolidation: %v", err)
	}
	return nil
}

func proformaCal(stockUpdates, currentStock map[string]map[string][]int) error {

	for productUID, colors := range stockUpdates {
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
	return nil
}

func saleConsolidation(stock, saleEntries []Proforma, currentStock map[string]map[string][]int) error {
	var finalStock []Proforma

	// Keep all existing entries except in_stock entries that were updated
	for _, item := range stock {
		if item.Type == "in_stock" {
			// Check if this product was updated in currentStock
			if updatedColors, exists := currentStock[item.Product.UID]; exists {
				// Calculate new total for updated stock
				total := 0
				for _, quantities := range updatedColors {
					for _, qty := range quantities {
						total += qty
					}
				}
				//fmt.Printf("Updating: %s\n colors: %v\n total %d\n", item.Product.UID, updatedColors, total)
				// Update the existing in_stock entry with new quantities
				updatedStockEntry := item
				updatedStockEntry.Product.Color = updatedColors
				updatedStockEntry.Product.Total = total
				finalStock = append(finalStock, updatedStockEntry)
			} else {
				// Keep unchanged in_stock entries as is
				finalStock = append(finalStock, item)
			}
		} else {
			// Keep all non-stock entries (purchases, sales, etc.)
			finalStock = append(finalStock, item)
		}
	}

	// Add new sale entries
	finalStock = append(finalStock, saleEntries...)

	// Write updated inventory back to file
	updatedData, err := json.MarshalIndent(finalStock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated inventory: %v", err)
	}

	if err := os.WriteFile("Data/inventory.json", updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated inventory: %v", err)
	}

	//fmt.Printf("\nSuccessfully updated inventory. Added %d sale entries and updated in_stock quantities.\n", len(saleEntries))
	return nil
}
