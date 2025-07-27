package pkg

import "fmt"

type StockStruct struct {
	ID       string
	Type     string
	Invoice  int
	Date     string
	Gst      string
	Design   string
	Color    colorStruct
	Total    int
	Rejected bool
	IsPaid   bool
	Gen      string
}

type colorStruct struct {
	black []int
	white []int
}

func Stock() {

	fmt.Println("Processing Stocks")
}
