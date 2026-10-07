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
Remaining-stock previews show the design and plain variants actually consumed;
sale history and invoices retain the requested design.

`update invoice` updates payments only. Stock quantities are updated through
`apply`.