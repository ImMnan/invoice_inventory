package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func GetPurchase(fileName string) ([]byte, error) {
	// Read the purchase data from the specified file
	piFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer piFile.Close()
	piData, err := csv.NewReader(piFile).ReadAll()
	if err != nil {
		return nil, err
	}
	var purchaseData []Purchase
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
			UUID:    row[0],
			Invoice: row[1],
			Product: ProductStruct{
				UID: strings.TrimSpace(row[2]), // Product_Id
				Gen: strings.TrimSpace(row[5]), // Gen
				//	GST:   strings.TrimSpace(row[6]), // GST as string
				Color: colorMap,
				Total: total,
			},
		})
	}

	return json.Marshal(purchaseData)
}
