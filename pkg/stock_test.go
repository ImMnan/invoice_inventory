package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStockUpdatesSeparateDesigns(t *testing.T) {
	products := ProductSlice{
		{Type: "purchase-invoice", Product: []ProductStruct{{ProductID: "TEE", Color: map[string][]int{"Red": {3}}, Quantity: 3}}},
		{Type: "purchase-invoice", Product: []ProductStruct{{ProductID: "TEE", Print: "Floral", Color: map[string][]int{"Red": {5}}, Quantity: 5}}},
		{Type: "proforma", Product: []ProductStruct{{ProductID: "TEE", Print: "Floral", Color: map[string][]int{"Red": {2}}, Quantity: 2}}},
		{Type: "purchase-invoice", Product: []ProductStruct{{ProductID: "OTHER", Print: "Floral", Color: map[string][]int{"Red": {7}}, Quantity: 7}}},
	}
	update, err := MakeStkUpdate(&products)
	if err != nil {
		t.Fatal(err)
	}
	if got := update.purchaseStkUpdates[StockKey("TEE", "plain")]["red"][0]; got != 3 {
		t.Fatalf("plain purchase = %d, want 3", got)
	}
	if got := update.purchaseStkUpdates[StockKey("TEE", "floral")]["red"][0]; got != 5 {
		t.Fatalf("floral purchase = %d, want 5", got)
	}
	if got := update.proformaStkUpdates[StockKey("TEE", "Floral")]["red"][0]; got != 2 {
		t.Fatalf("floral sale = %d, want 2", got)
	}
	if got := update.purchaseStkUpdates[StockKey("OTHER", "Floral")]["red"][0]; got != 7 {
		t.Fatalf("other product purchase = %d, want 7", got)
	}
	if update.PurchaseEntries[0].Product[0].Print != "plain" || update.PurchaseEntries[1].Product[0].Print != "Floral" {
		t.Fatal("purchase history lost print names")
	}
}

func TestPurchaseCSVPrint(t *testing.T) {
	for _, test := range []struct{ name, header, row, printName string }{
		{"legacy", "Type,Invoice,From,Product_Id,Date,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity", "purchase-invoice,1,vendor,TEE,today,Male,red,3,0,0,0,0,0,0,0,3", "plain"},
		{"design", "Type,Invoice,From,Product_Id,Date,Print,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity", "purchase-invoice,1,vendor,TEE,today,Floral,Male,red,3,0,0,0,0,0,0,0,3", "Floral"},
		{"blank", "Type,Invoice,From,Product_Id,Date,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity,Print", "purchase-invoice,1,vendor,TEE,today,Male,red,3,0,0,0,0,0,0,0,3, ", "plain"},
		{"print-first", "Print,Type,Invoice,From,Product_Id,Date,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity", "Floral,purchase-invoice,1,vendor,TEE,today,Male,red,3,0,0,0,0,0,0,0,3", "Floral"},
		{"proforma-no-print", "Type,Invoice,For,Product_Id,Date,Price,Gst,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity", "proforma,1,customer,TEE,today,100,5,Male,red,3,0,0,0,0,0,0,0,3", "plain"},
		{"proforma-design", "Type,Invoice,For,Product_Id,Date,Price,Gst,Print,Gen,Color,XS,S,M,L,XL,2XL,3XL,4XL,Quantity", "proforma,1,customer,TEE,today,100,5,Floral,Male,red,3,0,0,0,0,0,0,0,3", "Floral"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "purchase.csv")
			if err := os.WriteFile(path, []byte(test.header+"\n"+test.row+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			products, err := (&FileData{Data: path}).GetStockUpdate()
			if err != nil {
				t.Fatal(err)
			}
			product := products[0].Product[0]
			if product.Print != test.printName || product.Color["red"][0] != 3 {
				t.Fatalf("unexpected purchase product: %+v", product)
			}
		})
	}
}

func TestInventoryDesignIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(path, []byte(`[{"type":"in_stock","product":[{"product_id":"TEE","color":{"Red":[10,0,0,0,0,0,0,0]}}]}]`), 0600); err != nil {
		t.Fatal(err)
	}
	db := &JsLocalDB{InventoryFile: path}
	products := ProductSlice{{Type: "purchase-invoice", Product: []ProductStruct{{ProductID: "TEE", Print: "Floral", Color: map[string][]int{"red": {5, 0, 0, 0, 0, 0, 0, 0}}}}}}
	update, _ := MakeStkUpdate(&products)
	if err := db.UpdateInventoryFromStockUpdate(&update); err != nil {
		t.Fatal(err)
	}
	products = ProductSlice{{Type: "proforma", Product: []ProductStruct{{ProductID: "TEE", Print: "floral", Color: map[string][]int{"RED": {2, 0, 0, 0, 0, 0, 0, 0}}}}}}
	update, _ = MakeStkUpdate(&products)
	if err := db.UpdateInventoryFromStockUpdate(&update); err != nil {
		t.Fatal(err)
	}
	entries, stock, err := db.GetExistingStock()
	if err != nil {
		t.Fatal(err)
	}
	if stock[StockKey("TEE", "plain")]["red"][0] != 10 || stock[StockKey("TEE", "Floral")]["red"][0] != 3 {
		t.Fatalf("stock leaked between designs: %v", stock)
	}
	for _, entry := range entries {
		for _, product := range entry.Product {
			if product.Print == "" {
				encoded, _ := json.Marshal(entry)
				t.Fatalf("empty print persisted: %s", encoded)
			}
		}
	}
	products[0].Product[0].Color["RED"][0] = 4
	update, _ = MakeStkUpdate(&products)
	before, _ := os.ReadFile(path)
	if err := db.UpdateInventoryFromStockUpdate(&update); err == nil {
		t.Fatal("expected insufficient design stock error")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed update modified inventory")
	}
}

func TestPurchaseDesignAppendsToExistingStock(t *testing.T) {
	directory := t.TempDir()
	inventoryPath := filepath.Join(directory, "inventory.json")
	initial := `[
		{"uuid":"other","type":"in_stock","invoice":"OTHER","product":[{"product_id":"OTHER","print":"plain","color":{"black":[4]}}]},
		{"uuid":"1004","type":"in_stock","invoice":"NA","date":"2025-06-01","product":[{"product_id":"FC200G-RG","name":"Existing name","print":"Polo","gen":"unisex","gst":5,"price":250,"description":"Existing description","color":{"black":[10]},"quantity":10}]}
	]`
	if err := os.WriteFile(inventoryPath, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(directory, "products.json")
	if err := os.WriteFile(catalogPath, []byte(`[{"product_id":"FC200G-RG","name":"Catalog name","description":"Catalog description"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	db := &JsLocalDB{InventoryFile: inventoryPath, ProductFile: catalogPath}
	for _, purchase := range []struct {
		printName, color string
		quantity         int
	}{
		{"Calvin Klein", "black", 5},
		{"Calvin Klein", "blue", 3},
		{"calvin klein", "black", 2},
		{"", "black", 4},
	} {
		products := ProductSlice{{Type: "purchase-invoice", Invoice: "PO-1", Product: []ProductStruct{{
			ProductID: "FC200G-RG", Print: purchase.printName,
			Color: map[string][]int{purchase.color: {purchase.quantity}}, Quantity: purchase.quantity,
		}}}}
		update, err := MakeStkUpdate(&products)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.UpdateInventoryFromStockUpdate(&update); err != nil {
			t.Fatal(err)
		}
	}
	entries, _, err := db.GetExistingStock()
	if err != nil {
		t.Fatal(err)
	}
	stockCount, historyQuantity := 0, 0
	var variants map[string]ProductStruct
	for _, entry := range entries {
		if entry.Type == "purchase" {
			for _, product := range entry.Product {
				historyQuantity += product.Quantity
			}
		}
		if entry.Type != "in_stock" {
			continue
		}
		stockCount++
		if entry.UUID != "1004" {
			continue
		}
		if entry.Invoice != "NA" || entry.Date != "2025-06-01" || len(entry.Product) != 3 {
			t.Fatalf("existing stock container changed: %+v", entry)
		}
		variants = make(map[string]ProductStruct)
		for _, product := range entry.Product {
			if product.ProductID != "FC200G-RG" {
				t.Fatalf("design appended to wrong stock container: %+v", product)
			}
			variants[product.Print] = product
		}
	}
	if stockCount != 2 || historyQuantity != 14 || len(variants) != 3 {
		t.Fatalf("stock containers=%d, purchase quantity=%d, variants=%v", stockCount, historyQuantity, variants)
	}
	if variants["Polo"].Color["black"][0] != 10 || variants["Calvin Klein"].Color["black"][0] != 7 || variants["Calvin Klein"].Color["blue"][0] != 3 || variants["plain"].Color["black"][0] != 4 {
		t.Fatalf("unexpected design quantities: %+v", variants)
	}
	for _, printName := range []string{"Calvin Klein", "plain"} {
		product := variants[printName]
		if product.Name != "Catalog name" || product.Description != "Catalog description" || product.Price != 250 || product.GST != 5 || product.Gen != "unisex" {
			t.Fatalf("new design lost metadata: %+v", product)
		}
	}
}

func TestPurchaseNewProductStockContainer(t *testing.T) {
	for _, test := range []struct {
		name, inventory string
		productCount    int
	}{
		{"empty inventory", `[]`, 2},
		{"existing inventory", `[{"uuid":"1004","type":"in_stock","invoice":"NA","product":[{"product_id":"OTHER","print":"plain","color":{"red":[4]}}]}]`, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			inventoryPath := filepath.Join(directory, "inventory.json")
			catalogPath := filepath.Join(directory, "products.json")
			if err := os.WriteFile(inventoryPath, []byte(test.inventory), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(catalogPath, []byte(`[{"product_id":"TEE","name":"Catalog shirt","gen":"unisex","gst":5,"price":200,"description":"Cotton"}]`), 0600); err != nil {
				t.Fatal(err)
			}
			products := ProductSlice{{Type: "purchase-invoice", Invoice: "PO-2", Product: []ProductStruct{
				{ProductID: "TEE", Print: "Floral", Color: map[string][]int{"black": {3}}},
				{ProductID: "TEE", Print: "Stripes", Color: map[string][]int{"blue": {2}}},
			}}}
			update, err := MakeStkUpdate(&products)
			if err != nil {
				t.Fatal(err)
			}
			db := &JsLocalDB{InventoryFile: inventoryPath, ProductFile: catalogPath}
			if err := db.UpdateInventoryFromStockUpdate(&update); err != nil {
				t.Fatal(err)
			}
			entries, _, err := db.GetExistingStock()
			if err != nil {
				t.Fatal(err)
			}
			stockCount := 0
			for _, entry := range entries {
				if entry.Type != "in_stock" {
					continue
				}
				stockCount++
				if entry.UUID == "" || entry.Invoice != "NA" || len(entry.Product) != test.productCount {
					t.Fatalf("unexpected stock container: %+v", entry)
				}
				for _, product := range entry.Product {
					if product.ProductID == "TEE" && (product.Name != "Catalog shirt" || product.Gen != "unisex" || product.GST != 5 || product.Price != 200 || product.Description != "Cotton") {
						t.Fatalf("new product lost catalog metadata: %+v", product)
					}
				}
			}
			if stockCount != 1 {
				t.Fatalf("stock containers = %d, want 1", stockCount)
			}
		})
	}
}
