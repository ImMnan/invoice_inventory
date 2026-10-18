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

### Applying bank statements

Use `--hdfc` to explicitly select the HDFC transaction importer. Without a bank
flag, `apply` uses the existing sale/purchase stock/invoice importer; it does not
detect banks from CSV headers or file names.
Required HDFC columns are `Date`, `Narration`, `Debit Amount`, `Credit Amount`,
`Chq/Ref Number`, and `Closing Balance`. Headers are case-insensitive and
surrounding padding is ignored; extra columns such as `Value Dat` are ignored.

```sh
lvs apply -f transaction-oct.csv --hdfc
lvs apply -f transaction-oct.csv --hdfc --approve
lvs apply -f transaction-sep.csv --hdfc -m 9 --approve
```

Without `--approve`, this previews only new transactions, their assigned IDs,
and the selected destination. `--csv` produces the preview as CSV. Omitted
`-m` or `-m 0` selects `data.transactionData` (the configured working month),
not the calendar month or the month in the file name. `-m 1` through `-m 12`
selects the corresponding monthly section; `-m 13` selects `dataTest`.
Configure `transactionData` in each section you use. Missing configuration
is an error. A missing destination file is created on approval; its parent
directory must already exist. An existing destination must contain a JSON array
(`[]` is valid); malformed, empty, or `null` files are rejected.

Bank formats implement `pkg.TransactionStatement` through
`Parse(io.Reader) ([]pkg.BankTransaction, error)`. `pkg.TransactionHDFC` implements
the HDFC format. Future banks and financial institutions can implement the same
interface and add an explicit CLI flag, reusing the preview, duplicate detection,
ID assignment, and safe persistence pipeline. Only HDFC is currently supported.

Statement dates use `DD/MM/YY` and are stored as `YYYY-MM-DD`. Narration is kept
with surrounding padding removed. `UPI-` and `REV-UPI-` narrations use mode
`UPI`; cash narrations beginning with `CASH` use `CASH`; everything else uses
`NETB`. Imported `invoiceIds`, `label`, and `for` are empty arrays; `title` and
`notes` are empty strings. Amounts remain JSON numbers and retain exact cents;
blank debit/credit cells mean zero. Blank closing balances, fractional cents,
invalid amounts/dates, out-of-order rows, and discontinuous balances are errors.
In Go, transaction amounts use `pkg.Money`, an integer count of cents; use
`pkg.ParseMoney` for currency-unit input and `Money.String` for display.

Reference numbers retain leading zeros. All-zero references receive a stable
15-digit nonzero numeric ID. If distinct transactions reuse a nonzero reference
(for example, a payment and its reversal), the first retains the reference and
the next receives a deterministic suffix. Generated ID collisions reject the
import instead of silently replacing a transaction.

Repeat imports do not write the destination. Overlapping statements are matched
as an ordered sequence using date, narration, debit, credit, closing balance,
and reference where available; only the new suffix is appended. Previously
imported older subsets are skipped. If there is no overlap, a new-only statement
must continue from the destination's last date and balance. Conflicting or
ambiguous overlap is rejected; supply a statement with more surrounding rows
to establish continuity. Importing does not backfill or replace existing records.

Existing records, annotations, and additional JSON fields are preserved.
Approved imports write to a temporary file in the destination directory and
atomically replace the JSON array, preserving file permissions. A per-destination
`.lock` file prevents simultaneous LVS transaction imports, and the destination
is rechecked before replacement. If an interrupted process leaves a lock behind,
confirm no import is running before removing that specific lock and retrying.
No stock or invoice files are changed by a transaction import.