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

//func TestInterface(pf processFile) []TshirtStruct {
//	demo, _ := pf.processProductData()
//	return demo
//}

func (file *FileData) GetStockUpdate() (ProductSlice, error) {
	dataFile, err := os.Open(file.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}
	defer dataFile.Close()

	dataValue, err := csv.NewReader(dataFile).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv data: %w", err)
	}
	var productData []TshirtStruct

	for i, row := range dataValue {
		if i == 0 { // Skip header row
			continue
		}
		if row[0] == "proforma" { // Ensure we only process proforma rows

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

			productData = append(productData, TshirtStruct{
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
		} else if row[0] == "purchase-invoice" { // Ensure we only process purchase invoice rows

			if len(row) < 15 { // Ensure we have all required columns
				continue
			}
			// Parse size quantities
			colorMap := make(map[string][]int)
			quantities := make([]int, 8)
			// Parse quantities for each size (columns 6-13)
			for j := 0; j < 8; j++ {
				if qty, err := strconv.Atoi(strings.TrimSpace(row[6+j])); err == nil {
					quantities[j] = qty
				}
			}
			// Color is in column 5, use it as key for the map
			color := strings.TrimSpace(row[5])
			colorMap[color] = quantities
			// Parse total (column 14)
			total := 0
			if totalStr := strings.TrimSpace(row[14]); totalStr != "" {
				if t, err := strconv.Atoi(totalStr); err == nil {
					total = t
				}
			}

			productData = append(productData, TshirtStruct{
				UUID:    uuid.New().String(),
				Type:    strings.TrimSpace(row[0]),
				Invoice: strings.TrimSpace(row[1]),
				Date:    strings.TrimSpace(row[3]), // Date
				From:    "vendor",                  // Default vendor since it's not in CSV
				Product: ProductStruct{
					UID:   strings.TrimSpace(row[2]), // Product_Id
					Gen:   strings.TrimSpace(row[4]), // Gen
					Color: colorMap,
					Total: total,
				},
			})
		} else {
			return nil, fmt.Errorf("invalid row type %s at row %d", row[0], i)
		}

	}
	return productData, nil

}

func (manual ManualData) GetStockUpdate() (ProductSlice, error) {
	poData := manual.Data
	var productData []TshirtStruct

	for i, row := range poData {
		if row[0] == "proforma" && row[0] != "purchase-invoice" { // Ensure we only process proforma rows

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

			productData = append(productData, TshirtStruct{
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

		if row[0] == "purchase-invoice" && row[0] != "proforma" {
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

			productData = append(productData, TshirtStruct{
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

	}
	return productData, nil
}

func (data *JsLocalDB) getExistingStock() ([]In_stockTshirtStruct, map[string]map[string][]int, error) {
	inventoryDB, err := os.Open(data.File)
	if err != nil {
		return []In_stockTshirtStruct{}, nil, err
	}
	defer inventoryDB.Close()
	var allEntries []In_stockTshirtStruct
	if err := json.NewDecoder(inventoryDB).Decode(&allEntries); err != nil {
		return []In_stockTshirtStruct{}, nil, err
	}
	// Collect current stock state and apply subtractions
	currentStock := make(map[string]map[string][]int) // productUID -> color -> quantities

	// First, collect all existing in_stock data
	for _, item := range allEntries {
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
	return allEntries, currentStock, nil
}
