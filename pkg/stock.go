package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

type StockStruct struct {
	ID        string
	ProductID string
	Type      string
	Invoice   int
	Created   string
	Gst       string
	Design    string
	Color     colorStruct
	Total     int
	Rejected  bool
	IsPaid    bool
	Gen       string
}

type colorStruct struct {
	black []int
	white []int
}

func (s StockStruct) getStock() ([]byte, error) {
	// Implement your logic here, for now just return the ID
	return json.Marshal(s)
}

type Stock interface {
	getStock() ([]byte, error)
}

func Stocks() {
	// open and readh csv file
	csvFile, err := os.Open("Data/inventory.csv")
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV file:", err)
		return
	}

	for i, record := range records {
		if i == 0 {
			continue // skip header row
		}
		// Process each record as needed
		recordType := string(record[0])
		recInvoice := record[1]
		recProductID := record[2]
		recDate := record[3]
		recGst := record[4]
		recDesign := record[5]
		recColor := record[6]
		recXS := record[7]
		recS := record[8]
		recM := record[9]
		recL := record[10]
		recXL := record[11]
		recXXL := record[12]
		recXXXL := record[13]
		recXXXXL := record[14]
		recTotal := record[15]
		recRejected := record[16]
		recIsPaid := record[17]

		stock := StockStruct{
			ID:        fmt.Sprintf("ID-%d", i),
			ProductID: recProductID,
			Type:      recordType,
			Invoice:   recInvoice,
			Created:   recDate,
			Gst:       recGst,
			Design:    recDesign,
			Color: colorStruct{
				recColor: []int{recXS, recS, recM, recL, recXL, recXXL, recXXXL, recXXXXL},
			},
			Total:    recTotal,
			Rejected: recRejected,
			IsPaid:   recIsPaid,
		}

	}
}
