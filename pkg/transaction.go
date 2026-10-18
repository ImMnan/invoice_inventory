package pkg

import (
	"encoding/json"
	"io"
	"os"
)

func (data *JsLocalDB) Transactions() ([]byte, error) {

	var transaction []Transactions
	file, err := os.Open(data.TransactionFile)
	if err != nil {
		// handle error, e.g., log it or return
		return nil, err
	}
	defer file.Close()
	// You can add code here to read and unmarshal the transactions from the file
	fileData, err := io.ReadAll(file)
	if err != nil {
		// handle error, e.g., log it or return
		return nil, err
	}
	err = json.Unmarshal(fileData, &transaction)
	if err != nil {
		// handle error, e.g., log it or return
		return nil, err
	}

	transactionData, err := json.Marshal(transaction)
	if err != nil {
		// handle error, e.g., log it or return
		return nil, err
	}
	return transactionData, nil
}
