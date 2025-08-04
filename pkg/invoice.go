package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Invoice struct {
	Type    string        `json:"type"`
	For     string        `json:"for,omitempty"`
	Invoice string        `json:"invoice"`
	Date    string        `json:"date,omitempty"`
	IsPaid  bool          `json:"isPaid"`
	Product ProductStruct `json:"product"`
}

type ProductData struct {
	ProductId   string              `json:"product_id"`
	Name        string              `json:"name"`
	Colors      map[string][]string `json:"colors"`
	Description string              `json:"description"`
	Price       float64             `json:"price"`
}

type CustomerData struct {
	CustomerId string    `json:"customer_id"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	Rating     int       `json:"rating"`
	GstNumber  string    `json:"gst_number"`
	Contact    int       `json:"contact"`
	Email      string    `json:"email"`
	Origin     time.Time `json:"origin"`
	Country    string    `json:"country"`
}

func makeInvoice(stockUpdates map[string]map[string][]int) error {
	// Create a new invoice based on the stock updates data

	// Load products data for name and price lookup
	productsData, err := getProductData("Data/products.json")
	if err != nil {
		fmt.Println("Error fetching product data:", err)
		return err
	}

	// Create a map for quick product lookup by ID
	productMap := make(map[string]ProductData)
	for _, product := range productsData {
		productMap[product.ProductId] = product
	}

	// Print header
	fmt.Println("Product ID, Product Name, Price, Color, XS, S, M, L, XL, 2XL, 3XL, 4XL, Total, Amount")

	// Process each product in stockUpdates
	for productId, colors := range stockUpdates {
		// Look up product name and price
		productInfo, exists := productMap[productId]
		if !exists {
			return fmt.Errorf("error: Product ID %s not found in products.json\n", productId)
		}

		// Process each color for this product
		for color, quantities := range colors {
			// Calculate total quantity
			total := 0
			for _, qty := range quantities {
				total += qty
			}

			// Calculate amount (price × total)
			amount := productInfo.Price * float64(total)

			// Format the line as specified: Product ID, Product Name, Price, Color, XS, S, M, L, XL, 2XL, 3XL, 4XL, Total, Amount
			line := fmt.Sprintf("%s, %s, %.0f, %s",
				productId,
				productInfo.Name,
				productInfo.Price,
				color)

			// Add size quantities (XS, S, M, L, XL, 2XL, 3XL, 4XL)
			for _, qty := range quantities {
				line += fmt.Sprintf(", %d", qty)
			}

			// Add total and amount
			line += fmt.Sprintf(", %d, %.0f", total, amount)

			// Print for now (later we'll write to file)
			fmt.Println(line)
		}
	}

	// Following is the output I need to be passed to a txt file.
	// SHRIKRISHNA TECH, GST: 1234567890, India

	// Type, Invoice, Date
	// Invoice, SKCP-001, 2023-01-01

	// Name, Address, Gst Number, Country, Date
	// RK Tees, Navsari, 1234567890, India, 2023-01-01

	// Product ID, Product Name, Price, Color, XS, S, M, L, XL, 2XL, 3XL, 4XL, Total, Amount
	// CO180G-RG, 100% Cotton | T-Shirt | Bio-Washed, 200, Red, 1, 5, 5, 5, 5, 5, 1, 1, 28, 5600

	// if tax invoice
	// iGST, sGST, GstTotal
	// 2.5%, 2.5%, 5% of Amount (i.e 280)

	// TaxInvoice Total amount, Payment Terms, Payment Mode
	// 5880, Due on Receipt, Online Transfer/Cash
	return nil
}

func getProductData(fileName string) ([]ProductData, error) {

	filedata, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer filedata.Close()

	var product []ProductData
	err = json.NewDecoder(filedata).Decode(&product)
	if err != nil {
		return nil, err
	}
	return product, nil
}
