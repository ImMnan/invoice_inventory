# Inventory Prints

Stock is identified by product ID, print/design, color and size. Different designs
of the same product and color have independent quantities. Product IDs remain the
primary identity; designs do not replace them.

Purchase and proforma CSVs accept an optional `Print` column anywhere in the
header. Missing or blank print names default to `plain`. Existing blank-print
inventory is treated as `plain` and normalized when inventory is next saved.
Use the existing CSV templates for the remaining column names.

```sh
lvs get stock CO180G-RG --print Floral --color red
lvs get stock --print plain
lvs get stock --printed
lvs get purchase CO180G-RG --print Floral --color red
lvs get sale CO180G-RG --print Floral --color red
lvs apply --file purchase.csv --print Floral --color red
lvs apply --file purchase.csv --approve
```

Stock listings sort by product ID, then design, then color. Remaining-stock
previews sort by design, then color, then product ID. Print and color matching
is case-insensitive; `--printed` excludes `plain` stock.

The `apply` print/color flags filter previews only. `--approve` processes the
entire CSV, including all its designs and colors. New purchased designs create
separate product objects in the existing `in_stock.product` array for that
product ID. Catalog metadata is inherited, with existing stock metadata as a
fallback for fields absent from the catalog. Purchases of an existing design
update its color quantities without duplicating the product object. Purchase
history remains in separate `purchase` records. An `in_stock` record is created
only when none exists.

Sales consume the requested design first, then cover any shortage from `plain`
stock of the same product ID, color and size. This also works when the design or
its color is missing. Other printed designs are never used as substitutes, and
plain sales consume only plain stock. Plain sales are allocated before design
fallbacks. Insufficient combined stock rejects the update without saving it.
Remaining-stock previews for both sales and purchases show every color and
design (including `plain`) for each product ID affected by the preview, not
just the variants consumed or purchased. Quantities are accumulated across
existing stock records, and size totals and the grand total include all displayed
variants. Print/color filters select the invoice lines used for the preview;
the remaining-stock summary still includes all variants of those products.
Unrelated product IDs are excluded. Sale history and invoices retain the
requested design.

`update invoice` updates payments only. Stock quantities are updated through
`apply`.

## Transactions

Configure `transactionData` under `data` (or the selected `dataJan` through
`dataDec` section) in your LVS configuration to point to the transaction JSON.
Transaction records use arrays for `invoiceIds`, `label`, and `for`.

```sh
lvs get transaction
lvs get transaction SK25CPQ10303 --label sales --for Ankit
lvs get transaction -l sales -f Ankit --csv
lvs get transaction --show-labels
lvs get transaction --for Ankit --show-labels --csv
```

Invoice, label, and recipient filters combine with AND. Invoice IDs match
exactly; labels and recipients use case-insensitive exact matching with
surrounding whitespace ignored. Any matching array entry satisfies its filter.
`--show-labels` adds a `LABEL` column to both table and CSV output, showing each
transaction's nonblank labels separated by commas in their stored order.
Transactions without labels have an empty label cell. Without the flag, the
column is omitted. `--csv` produces CSV including a header.
Filtering preserves each transaction's
stored closing balance; it does not recalculate balances.