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

func TestRemainingStockIncludesAllProductVariants(t *testing.T) {
	original := `[
			{"type":"in_stock","product":[
				{"product_id":"CO180G-RG","color":{"Red":[10,20],"blue":[4,6]}},
				{"product_id":"CO180G-RG","print":"Floral","color":{"red":[5,7]}},
				{"product_id":"CO180G-RG","print":"Stripes","color":{"green":[8,9]}},
				{"product_id":"OTHER","color":{"red":[99,99]}}
			]},
			{"type":"in_stock","product":[
				{"product_id":"CO180G-RG","print":"plain","color":{"RED":[2,3]}}
			]},
			{"type":"purchase","product":[
				{"product_id":"CO180G-RG","print":"Historical","color":{"red":[100,100]}}
			]}
		]`
	product := func(id, printName, color string, quantities ...int) pkg.ProductStruct {
		return pkg.ProductStruct{ProductID: id, Print: printName, Color: map[string][]int{color: quantities}}
	}
	row := func(id, printName, color, xs, s, total string) []string {
		return []string{id, printName, color, xs, s, "0", "0", "0", "0", "0", "0", total}
	}
	for _, test := range []struct {
		name                     string
		isSale                   bool
		colorFilter, printFilter string
		products                 []pkg.ProductStruct
		want                     [][]string
	}{
		{
			name: "sale across invoices", isSale: true,
			products: []pkg.ProductStruct{
				product("CO180G-RG", "Floral", "red", 2, 3),
				product("CO180G-RG", "Floral", "red", 1, 1),
				product("JOB_SERVICE", "", "red", 1),
				product("TRADE-ML-ITEM", "", "red", 1),
			},
			want: [][]string{
				row("CO180G-RG", "Floral", "red", "2", "3", "5"),
				row("CO180G-RG", "plain", "blue", "4", "6", "10"),
				row("CO180G-RG", "plain", "red", "12", "23", "35"),
				row("CO180G-RG", "Stripes", "green", "8", "9", "17"),
				row("GRAND TOTAL", "", "", "26", "41", "67"),
			},
		},
		{
			name: "purchase", products: []pkg.ProductStruct{product("CO180G-RG", "Floral", "red", 3, 4)},
			want: [][]string{
				row("CO180G-RG", "Floral", "red", "8", "11", "19"),
				row("CO180G-RG", "plain", "blue", "4", "6", "10"),
				row("CO180G-RG", "plain", "red", "12", "23", "35"),
				row("CO180G-RG", "Stripes", "green", "8", "9", "17"),
				row("GRAND TOTAL", "", "", "32", "49", "81"),
			},
		},
		{
			name: "purchase new design and color", products: []pkg.ProductStruct{product("CO180G-RG", "Dots", " Black ", 3, 4)},
			want: [][]string{
				row("CO180G-RG", "Dots", "black", "3", "4", "7"),
				row("CO180G-RG", "Floral", "red", "5", "7", "12"),
				row("CO180G-RG", "plain", "blue", "4", "6", "10"),
				row("CO180G-RG", "plain", "red", "12", "23", "35"),
				row("CO180G-RG", "Stripes", "green", "8", "9", "17"),
				row("GRAND TOTAL", "", "", "32", "49", "81"),
			},
		},
		{
			name: "purchase new product", products: []pkg.ProductStruct{product("NEW", "", "red", 3, 4)},
			want: [][]string{
				row("NEW", "plain", "red", "3", "4", "7"),
				row("GRAND TOTAL", "", "", "3", "4", "7"),
			},
		},
		{
			name: "filters select invoice lines but show full product", isSale: true,
			colorFilter: "RED", printFilter: "floral",
			products: []pkg.ProductStruct{
				product("CO180G-RG", "Floral", "red", 3, 4),
				product("CO180G-RG", "", "blue", 1, 1),
				product("OTHER", "Stripes", "red", 1, 1),
				product("OTHER", "Floral", "blue", 1, 1),
			},
			want: [][]string{
				row("CO180G-RG", "Floral", "red", "2", "3", "5"),
				row("CO180G-RG", "plain", "blue", "4", "6", "10"),
				row("CO180G-RG", "plain", "red", "12", "23", "35"),
				row("CO180G-RG", "Stripes", "green", "8", "9", "17"),
				row("GRAND TOTAL", "", "", "26", "41", "67"),
			},
		},
		{
			name: "no matching invoice lines", isSale: true, colorFilter: "green",
			products: []pkg.ProductStruct{product("CO180G-RG", "Floral", "red", 3, 4)},
			want:     [][]string{row("GRAND TOTAL", "", "", "0", "0", "0")},
		},
	} {
		for _, formatCsv := range []bool{true, false} {
			format := "table"
			if formatCsv {
				format = "csv"
			}
			t.Run(test.name+"/"+format, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "inventory.json")
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
				var entries []pkg.Proforma
				for _, product := range test.products {
					entries = append(entries, pkg.Proforma{Product: []pkg.ProductStruct{product}})
				}
				update := pkg.StockUpdate{PurchaseEntries: entries}
				if test.isSale {
					update = pkg.StockUpdate{SaleEntries: entries}
				}
				output := captureRemainingStock(t, &pkg.JsLocalDB{InventoryFile: path}, update, test.isSale, test.colorFilter, test.printFilter, formatCsv)
				summary := strings.SplitN(output, "\n\n", 2)[0]
				var rows [][]string
				if formatCsv {
					var err error
					rows, err = csv.NewReader(strings.NewReader(summary)).ReadAll()
					if err != nil {
						t.Fatal(err)
					}
				} else {
					for _, line := range strings.Split(strings.TrimSpace(summary), "\n") {
						if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
							continue
						}
						rows = append(rows, strings.Fields(line))
					}
				}
				if len(rows) != len(test.want)+1 {
					t.Fatalf("unexpected row count: %s", output)
				}
				for i, want := range test.want {
					got := rows[i+1]
					if formatCsv && len(got) != 12 {
						t.Fatalf("expected 12 columns, got %v", got)
					}
					if strings.Join(strings.Fields(strings.Join(got, " ")), " ") != strings.Join(strings.Fields(strings.Join(want, " ")), " ") {
						t.Errorf("row %d: got %v, want %v", i+1, got, want)
					}
				}
				stored, err := os.ReadFile(path)
				if err != nil || string(stored) != original {
					t.Fatal("preview changed inventory")
				}
			})
		}
	}
}

func captureRemainingStock(t *testing.T, db *pkg.JsLocalDB, update pkg.StockUpdate, isSale bool, colorFilter, printFilter string, formatCsv bool) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	stdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = stdout }()
	err = displayRemainingStock(db, update, isSale, colorFilter, printFilter, formatCsv)
	writer.Close()
	os.Stdout = stdout
	output, readErr := io.ReadAll(reader)
	if err != nil || readErr != nil {
		t.Fatalf("preview errors: %v, %v", err, readErr)
	}
	return string(output)
}

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
		{"sale", "proforma", "Floral", "Floral", "3", "10", 2, true},
		{"purchase", "purchase-invoice", "Floral", "Floral", "7", "10", 2, false},
		{"missing design", "proforma", "Stripes", "Floral", "5", "8", 2, true},
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
