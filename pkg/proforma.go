package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

func Proforma(fileName string) ([]byte, error) {

	poFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer poFile.Close()
	poData, err := csv.NewReader(poFile).ReadAll()
	if err != nil {
		return nil, err
	}

	var proformaData []TshirtStruct
	for i, row := range poData {
		if i == 0 { // Skip header row
			continue
		}
		if len(row) < 6 {
			continue
		}
		proformaData = append(proformaData, TshirtStruct{
			UUID:     row[0],
			Type:     row[1],
			Invoice:  row[2],
			Date:     row[3],
			IsPaid:   row[4] == "false",
			Rejected: row[5] == "false",
			Product: ProductStruct{
				UID:   row[6],
				Print: row[7],
				Gen:   row[8],
				GST:   row[9],
				Color: make(map[string][]int),
				Total: 0,
			},
		})
	}
	return json.Marshal(proformaData)
}

func ApplyProforma(fileName string) {
	proformaData, err := Proforma(fileName)
	if err != nil {
		// Handle error
		return
	}
	fmt.Print(string(proformaData))
}
