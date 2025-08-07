package pkg

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

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
