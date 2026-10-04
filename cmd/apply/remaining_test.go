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
		name, rowType, quantity string
		isSale                  bool
	}{
		{"sale", "proforma", "3", true},
		{"purchase", "purchase-invoice", "7", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			products := pkg.ProductSlice{{Type: test.rowType, Product: []pkg.ProductStruct{{ProductID: "TEE", Print: "Floral", Color: map[string][]int{"red": {2}}}}}}
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
			err = displayRemainingStock(&pkg.JsLocalDB{InventoryFile: path}, update, test.isSale, "RED", "floral", true)
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
			if len(rows) != 3 || rows[1][1] != "Floral" || rows[1][3] != test.quantity || rows[1][11] != test.quantity {
				t.Fatalf("unexpected remaining stock: %s", output)
			}
			stored, err := os.ReadFile(path)
			if err != nil || string(stored) != original {
				t.Fatal("preview changed inventory")
			}
		})
	}
}
