package pkg

type Proforma struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	Invoice  string        `json:"invoice"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

type Purchase struct {
	UUID    string        `json:"uuid"`
	Type    string        `json:"type"`
	Invoice string        `json:"invoice"`
	Date    string        `json:"date,omitempty"`
	Product ProductStruct `json:"product"`
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

type TshirtStruct struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	Invoice  string        `json:"invoice"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

