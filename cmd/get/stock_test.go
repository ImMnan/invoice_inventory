package get

import "testing"

func TestStockDesignFilter(t *testing.T) {
	filter := StockFilter{ProductID: "TEE", PrintFlag: "floral", ColorFlag: "RED"}
	if !filter.shouldShowStockProduct(Stocks{Type: "in_stock"}, ProductStruct{ProductID: "TEE"}) ||
		!filter.shouldShowPrintedProduct(ProductStruct{Print: "Floral"}) || !filter.shouldShowColor("red") {
		t.Fatal("matching product, design and color were excluded")
	}
	if filter.shouldShowPrintedProduct(ProductStruct{}) || filter.shouldShowStockProduct(Stocks{Type: "in_stock"}, ProductStruct{ProductID: "OTHER"}) {
		t.Fatal("nonmatching product or design was included")
	}
	if (StockFilter{Printed: true}).shouldShowPrintedProduct(ProductStruct{Print: "plain"}) {
		t.Fatal("plain stock should not count as printed")
	}
	if !(StockFilter{PrintFlag: "plain"}).shouldShowPrintedProduct(ProductStruct{}) {
		t.Fatal("legacy empty print should match plain")
	}
}

func TestStockRowsSortByProductThenDesignThenColor(t *testing.T) {
	rows := []stockRow{
		{product: ProductStruct{ProductID: "A", Print: "Zebra"}, color: "blue"},
		{product: ProductStruct{ProductID: "A", Print: "Floral"}, color: "red"},
		{product: ProductStruct{ProductID: "Z", Print: "Floral"}, color: "blue"},
		{product: ProductStruct{ProductID: "B", Print: "Floral"}, color: "blue"},
		{product: ProductStruct{ProductID: "A", Print: "Floral"}, color: "blue"},
		{product: ProductStruct{ProductID: "A", Print: "Zebra"}, color: "red"},
	}
	sortStockRows(rows)
	want := []struct{ productID, design, color string }{
		{"A", "Floral", "blue"},
		{"A", "Floral", "red"},
		{"A", "Zebra", "blue"},
		{"A", "Zebra", "red"},
		{"B", "Floral", "blue"},
		{"Z", "Floral", "blue"},
	}
	for index, expected := range want {
		row := rows[index]
		if row.product.ProductID != expected.productID || row.product.Print != expected.design || row.color != expected.color {
			t.Fatalf("row %d = %s/%s/%s, want %s/%s/%s", index,
				row.product.ProductID, row.product.Print, row.color,
				expected.productID, expected.design, expected.color)
		}
	}
}
