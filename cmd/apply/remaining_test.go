package apply

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/immnan/invoice_invoice/pkg"
)

func TestRemainingStockDesignIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	original := `[{"type":"in_stock","product":[{"product_id":"TEE","color":{"Red":[10]}},{"product_id":"TEE","print":"Floral","color":{"red":[5]}}]}]`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, rowType, printName, rowPrint, quantity, plainQuantity string
		requested                                                   int
		isSale                                                      bool
	}{
		{"sale", "proforma", "Floral", "Floral", "3", "", 2, true},
		{"purchase", "purchase-invoice", "Floral", "Floral", "7", "", 2, false},
		{"missing design", "proforma", "Stripes", "plain", "8", "", 2, true},
		{"design shortage", "proforma", "Floral", "Floral", "0", "8", 7, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			products := pkg.ProductSlice{{Type: test.rowType, Product: []pkg.ProductStruct{{ProductID: "TEE", Print: test.printName, Color: map[string][]int{"red": {test.requested}}}}}}
			update, err := pkg.MakeStkUpdate(&products)
			if err != nil {
				t.Fatal(err)
			}
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout := os.Stdout
			os.Stdout = writer
			defer func() { os.Stdout = stdout }()
			err = displayRemainingStock(&pkg.JsLocalDB{InventoryFile: path}, update, test.isSale, "RED", test.printName, true)
			writer.Close()
			os.Stdout = stdout
			output, readErr := io.ReadAll(reader)
			reader.Close()
			if err != nil || readErr != nil {
				t.Fatalf("preview errors: %v, %v", err, readErr)
			}
			lines := strings.SplitN(string(output), "\n\n", 2)
			rows, err := csv.NewReader(strings.NewReader(lines[0])).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			wantRows := 3
			if test.plainQuantity != "" {
				wantRows = 4
			}
			if len(rows) != wantRows || rows[1][1] != test.rowPrint || rows[1][3] != test.quantity || rows[1][11] != test.quantity {
				t.Fatalf("unexpected remaining stock: %s", output)
			}
			if test.plainQuantity != "" && (rows[2][1] != "plain" || rows[2][3] != test.plainQuantity) {
				t.Fatalf("unexpected plain remaining stock: %s", output)
			}
			stored, err := os.ReadFile(path)
			if err != nil || string(stored) != original {
				t.Fatal("preview changed inventory")
			}
		})
	}
}
