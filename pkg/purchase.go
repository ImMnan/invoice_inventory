package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func (p PurBool) processFileData(data [][]string) ([]byte, error) {
	if p.active {
		var purchaseData []Purchase
		piData := data
		for i, row := range piData {
			if row[0] != "purchase-invoice" { // Ensure we only process purchase rows
				return nil, fmt.Errorf("invalid purchase data: expected 'purchase' type in first column")
			}
			if i == 0 { // Skip header row
				continue
			}
			if len(row) < 15 { // Ensure we have all required columns
				continue
			}

			// Parse size quantities
			colorMap := make(map[string][]int)
			quantities := make([]int, 8)
			// Parse quantities for each size (columns 8-15)
			for j := 0; j < 8; j++ {
				if qty, err := strconv.Atoi(strings.TrimSpace(row[6+j])); err == nil {
					quantities[j] = qty
				}
			}

			// Color is in column 6, use it as key for the map
			color := strings.TrimSpace(row[5])
			colorMap[color] = quantities

			// Parse total (column 16)
			total := 0
			if totalStr := strings.TrimSpace(row[14]); totalStr != "" {
				if t, err := strconv.Atoi(totalStr); err == nil {
					total = t
				}
			}

			purchaseData = append(purchaseData, Purchase{
				UUID:    uuid.New().String(),
				Type:    "purchase",
				Invoice: row[1],
				Date:    strings.TrimSpace(row[3]), // Date
				From:    strings.TrimSpace(row[2]), // Vendor ID
				Product: ProductStruct{
					UID:   strings.TrimSpace(row[2]), // Product_Id
					Gen:   strings.TrimSpace(row[5]), // Gen
					Color: colorMap,
					Total: total,
				},
			})
		}
		return json.Marshal(purchaseData)
	}
	return nil, nil
}

func (p *PurBool) ApplyFileData(data [][]string, format string) error {

	purchaseData, err := p.processFileData(data)
	if err != nil {
		return fmt.Errorf("\nerror processing purchase data:\n %v", err)
	}

	var purchaseItems []Purchase
	if err := json.Unmarshal(purchaseData, &purchaseItems); err != nil {
		return fmt.Errorf("\nerror unmarshalling purchase data:\n %v", err)
	}

	invoiceGroups := make(map[string][]Purchase)
	for _, item := range purchaseItems {
		invoiceID := item.Invoice
		invoiceGroups[invoiceID] = append(invoiceGroups[invoiceID], item)
	}

	for invoiceID, invoiceItems := range invoiceGroups {
		var vendorId string
		if len(invoiceItems) > 0 {
			vendorId = invoiceItems[0].From // considering 'From' is the vendor ID
		}
		if vendorId != "" {
			fmt.Printf("warning: no vendor id found for the invoice %s, skipping\n", invoiceID)
			continue
		}
		if err := addPurchase(invoiceItems); err != nil {
			return fmt.Errorf("\nerror adding purchase items for invoice %s:\n %v", invoiceID, err)
		}

	}

	return nil
}

func addPurchase(purchaseItems []Purchase) error {
	// This function should implement the logic to add purchase items to the database which is a json file.

	inventory, err := os.Open("Data/Inventory.json")
	if err != nil {
		return fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer inventory.Close()

	var stock []Purchase
	if err := json.NewDecoder(inventory).Decode(&stock); err != nil {
		return fmt.Errorf("failed to decode inventory data: %w", err)
	}

	var purchaseEntries []Purchase
	stockUpdates := make(map[string]map[string][]int) // productUID -> color -> quantities
	for _, purchaseItem := range purchaseItems {
		purchaseEntry := Purchase{
			UUID:    purchaseItem.UUID,
			Type:    purchaseItem.Type,
			From:    purchaseItem.From, // Copy vendor ID
			Invoice: purchaseItem.Invoice,
			Date:    purchaseItem.Date,
			Product: ProductStruct{
				UID:   purchaseItem.Product.UID,
				Gen:   purchaseItem.Product.Gen,
				Color: make(map[string][]int),
				Total: purchaseItem.Product.Total,
			},
		}
		//		purchaseEntries = append(purchaseEntries, purchaseEntry)
		for color, quantities := range purchaseItem.Product.Color {
			purchaseEntry.Product.Color[strings.ToLower(color)] = quantities
		}

		purchaseEntries = append(purchaseEntries, purchaseEntry)

		if stockUpdates[purchaseItem.Product.UID] == nil {
			stockUpdates[purchaseItem.Product.UID] = make(map[string][]int)
		}

		for color, quantities := range purchaseItem.Product.Color {
			colorKey := strings.ToLower(color)
			if existing, exists := stockUpdates[purchaseItem.Product.UID][colorKey]; exists {
				for i, qty := range quantities {
					for i < len(existing) {
						stockUpdates[purchaseItem.Product.UID][colorKey][i] += qty
					}
				}

			} else {
				//initialize with current quantities
				stockUpdates[purchaseItem.Product.UID][colorKey] = make([]int, len(quantities))
				copy(stockUpdates[purchaseItem.Product.UID][colorKey], quantities)
			}

		}
	}
	currentStock := make(map[string]map[string][]int)
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

	// Apply additions to the inventory

	if err := purchaseCal(stockUpdates, currentStock); err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	if err := purchaseConsolidation(stock, purchaseEntries, currentStock); err != nil {
		return fmt.Errorf("failed to consolidate stock: %w", err)
	}
	return nil
}

func purchaseCal(stockUpdates, currentStock map[string]map[string][]int) error {

	return nil

}

func purchaseConsolidation(stock, purchaseEntries []Purchase, currentStock map[string]map[string][]int) error {

	return nil

}

type ProductSlice []TshirtStruct

//type ProformaSlice []TshirtStruct
//type PurchaseSlice []TshirtStruct

type FileData struct {
	Data string
	// OsData *os.File
}

type ManualData struct {
	Data [][]string
}

type processFile interface {
	ProcessProductData() (ProductSlice, error)
}

//func TestInterface(pf processFile) []TshirtStruct {
//	demo, _ := pf.processProductData()
//	return demo
//}

func (file FileData) ProcessProductData() (ProductSlice, error) {
	dataFile, err := os.Open(file.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}
	defer dataFile.Close()

	dataValue, err := csv.NewReader(dataFile).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv data: %w", err)
	}
	poData := dataValue
	var proformaData []TshirtStruct
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

		proformaData = append(proformaData, TshirtStruct{
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
	return proformaData, nil
}

func (manual ManualData) ProcessProductData() (ProductSlice, error) {
	poData := manual.Data
	var proformaData []TshirtStruct
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

		proformaData = append(proformaData, TshirtStruct{
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
	return proformaData, nil
}
