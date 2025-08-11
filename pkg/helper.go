package pkg

type In_stockTshirtStruct struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	From     string        `json:"from,omitempty"`
	Invoice  string        `json:"invoice"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

type TshirtStruct struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	From     string        `json:"from,omitempty"`
	Invoice  string        `json:"invoice"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

type ProductStruct struct {
	UID   string           `json:"uid"`
	Print string           `json:"print"`
	Gen   string           `json:"gen"`
	GST   string           `json:"gst"`
	Color map[string][]int `json:"color"`
	Total int              `json:"total"`
	Price int              `json:"price"`
}

type ProductSlice []TshirtStruct

type Proforma = TshirtStruct
type Purchase = TshirtStruct

//type ProformaSlice []TshirtStruct
//type PurchaseSlice []TshirtStruct

type FileData struct {
	Data string
	// OsData *os.File
}

type ManualData struct {
	Data [][]string
}

type JsLocalDB struct {
	File string
}
type StockData interface {
	existingStock() (map[string]map[string][]int, error)
	updateInventoryFromStockUpdate(StockUpdate) (InvoiceGroupedData, error)
}

type MakeData interface {
	addProforma() (map[string]map[string][]int, []Proforma, error)
	addPurchase() (map[string]map[string][]int, []Purchase, error)
}

type StockUpdate struct {
	proformaStkUpdates map[string]map[string][]int
	purchaseStkUpdates map[string]map[string][]int
	SaleEntries        []Proforma
	PurchaseEntries    []Purchase
}

// InvoiceGroupedData represents the data grouped by invoice ID for invoice generation
type InvoiceGroupedData struct {
	SalesByInvoice        map[string][]Proforma                  // invoiceID -> sale entries (for invoice generation)
	StockChangesByInvoice map[string]map[string]map[string][]int // invoiceID -> productUID -> color -> quantities (for optimization)
}

func MakeStkUpdate(mkd MakeData) (StockUpdate, error) {
	// Process inventory updates and generate invoices for each invoice group
	proformaStkUpdates, saleEntries, err := mkd.addProforma()
	if err != nil {
		return StockUpdate{}, err
	}
	purchaseStkUpdates, purchaseEntries, err := mkd.addPurchase()
	if err != nil {
		return StockUpdate{}, err
	}

	// Create a StockUpdate instance to hold both updates and entries
	stockUpdate := StockUpdate{
		proformaStkUpdates: proformaStkUpdates,
		purchaseStkUpdates: purchaseStkUpdates,
		SaleEntries:        saleEntries,
		PurchaseEntries:    purchaseEntries,
	}
	return stockUpdate, nil
}
