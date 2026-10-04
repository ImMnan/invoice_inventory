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
	if len(dataValue) == 0 {
		return nil, fmt.Errorf("empty csv data")
	}
	headers := make(map[string]int)
	for index, header := range dataValue[0] {
		headers[strings.ToLower(strings.TrimSpace(header))] = index
	}
	typeColumn, exists := headers["type"]
	if !exists {
		return nil, fmt.Errorf("missing csv column %q", "type")
	}

	for i, row := range dataValue {
		if i == 0 { // Skip header row
			continue
		}
		rowType := strings.TrimSpace(row[typeColumn])
		if rowType != "proforma" && rowType != "purchase-invoice" {
			return nil, fmt.Errorf("invalid row type %s at row %d", rowType, i+1)
		}
		columns := []string{"type", "invoice", "for", "product_id", "date", "price", "gst", "print", "gen", "color", "xs", "s", "m", "l", "xl", "2xl", "3xl", "4xl", "quantity"}
		if rowType == "purchase-invoice" {
			columns = []string{"type", "invoice", "from", "product_id", "date", "gen", "color", "xs", "s", "m", "l", "xl", "2xl", "3xl", "4xl", "quantity", "print"}
		}
		ordered := make([]string, len(columns))
		for index, column := range columns {
			position, exists := headers[column]
			if !exists {
				if column == "print" {
					ordered[index] = "plain"
					continue
				}
				return nil, fmt.Errorf("missing csv column %q", column)
			}
			ordered[index] = strings.TrimSpace(row[position])
		}
		row = ordered
		if row[0] == "proforma" { // Ensure we only process proforma rows

			if len(row) < 19 { // Ensure we have all required columns
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

			// Parse total (column 14)
			//total := 0
			//if totalStr := strings.TrimSpace(row[14]); totalStr != "" {
			//	if t, err := strconv.Atoi(totalStr); err == nil {
			//		total = t
			//	}
			//}
			quantity := 0
			if qtyStr := strings.TrimSpace(row[18]); qtyStr != "" {
				if t, err := strconv.Atoi(qtyStr); err == nil {
					quantity = t
				}
			}
			priceInt := 0
			if p, err := strconv.Atoi(strings.TrimSpace(row[5])); err == nil {
				priceInt = p
			}
			gstInt := 0
			if g, err := strconv.Atoi(strings.TrimSpace(row[6])); err == nil {
				gstInt = g
			}
			total := quantity * priceInt
			productData = append(productData, TshirtStruct{
				UUID:     uuid.New().String(),
				Type:     strings.TrimSpace(row[0]),
				Invoice:  strings.TrimSpace(row[1]),
				For:      strings.TrimSpace(row[2]), // For field
				Date:     strings.TrimSpace(row[4]),
				IsPaid:   false, // Default to false for proforma
				Rejected: false, // Default to false for proforma
				Product: []ProductStruct{{
					ProductID: strings.TrimSpace(row[3]), // Product_Id
					Print:     NormalizePrint(row[7]),    // Print
					Gen:       strings.TrimSpace(row[8]), // Gen
					Price:     priceInt,
					GST:       gstInt,
					Color:     colorMap,
					Total:     total,
					Quantity:  quantity,
				}},
			})
		} else if row[0] == "purchase-invoice" { // Ensure we only process purchase invoice rows

			if len(row) < 17 { // Ensure we have all required columns
				continue
			}
			// Parse size quantities
			colorMap := make(map[string][]int)
			quantities := make([]int, 8)
			// Parse quantities for each size (columns 6-13)
			for j := 0; j < 8; j++ {
				if qty, err := strconv.Atoi(strings.TrimSpace(row[7+j])); err == nil {
					quantities[j] = qty
				}
			}
			// Color is in column 5, use it as key for the map
			color := strings.TrimSpace(row[6])
			colorMap[color] = quantities
			quantity := 0
			if qtyStr := strings.TrimSpace(row[15]); qtyStr != "" {
				if t, err := strconv.Atoi(qtyStr); err == nil {
					quantity = t
				}
			}

			priceInt := 0
			if p, err := strconv.Atoi(strings.TrimSpace(row[6])); err == nil {
				priceInt = p
			}

			total := quantity * priceInt
			productData = append(productData, TshirtStruct{
				UUID:    uuid.New().String(),
				Type:    strings.TrimSpace(row[0]),
				Invoice: strings.TrimSpace(row[1]),
				Date:    strings.TrimSpace(row[4]), // Date
				From:    strings.TrimSpace(row[2]), // Default vendor since it's not in CSV
				Product: []ProductStruct{{
					ProductID: strings.TrimSpace(row[3]), // Product_Id
					Print:     NormalizePrint(row[16]),
					Gen:       strings.TrimSpace(row[5]), // Gen
					Color:     colorMap,
					Total:     total,
					Quantity:  quantity,
				}},
			})

		} else {
			return nil, fmt.Errorf("invalid row type %s at row %d", row[0], i)
		}
	}
	return productData, nil

}

func (data *JsLocalDB) getExistingStock() ([]In_stockTshirtStruct, map[string]map[string][]int, error) {
	inventoryDB, err := os.Open(data.InventoryFile)
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
			for _, prod := range item.Product {
				stockKey := StockKey(prod.ProductID, prod.Print)
				if currentStock[stockKey] == nil {
					currentStock[stockKey] = make(map[string][]int)
				}
				for color, quantities := range prod.Color {
					color = strings.ToLower(strings.TrimSpace(color))
					existing := currentStock[stockKey][color]
					sizeCount := max(8, len(quantities))
					if len(existing) < sizeCount {
						padded := make([]int, sizeCount)
						copy(padded, existing)
						existing = padded
					}
					for index, quantity := range quantities {
						existing[index] += quantity
					}
					currentStock[stockKey][color] = existing
				}
			}
		}
	}
	return allEntries, currentStock, nil
}
