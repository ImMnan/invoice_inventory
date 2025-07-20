
The plan of execution

Type : [Purchase_Order, Purchase_Invoice, Proforma_Invoice, Sales_Invoice, In_Stock]
Product: [CO180G-RG, CO220G-RL, CO240G-RL, COLC220G-RG]
Color: [black, white, red, blue, yellow]
Size: [XS, S, M, L, XL, 2XL, 3XL, 4XL]
Invoice: Unique_id (serialised)
Data: Date of invoice
GST: string
isPaid: bool


Entry: 
```json
{
  "type": "Sales_Invoice",
  "invoice": "SK25CPQ10002",
  "product_id": "CO240G-RL",
  "gen": "men/women/unisex",
  "date": "2025-07-16",
  "gst": "0%",
  "isPaid": false,
  "Design": "Hades",
  "rejected": false, 
  "color": {
    "black": [1,1,1,1,1,1,1,1],
    "red":   [1,1,1,1,1,1,1,1]
    },
  "total": 16 
}
```

Product:
```json
{
  "type": "Product",
  "product_id": "CO240G-RL",
  "created": "2025-07-16",
  "gst": "0%",
  "Design": "<Design_name>",
  "color": [
    "black",
    "red",
    "White",
    "Blue"
  ],
  "from": ["Vendor", "vendor2"],
  "salePrice": 200  
}

```


```sh
lvs get stock
lvs get stock <product_id>
lvs get stock <product_id> --printed
lvs get stock <product_id> --csv
lvs get stock <product_id> --rejected/-r
lvs get stock <product_id> --color/-c <color> --size/-s <size_name>
lvs get invoice <invoice_id>
lvs get entry <entry_id>
lvs get product <product_id>
```

```sh
lvs apply -f <file_name> 
lvs apply -f <file_name> --approve 
```

```sh
lvs delete invoice <invoice_id> --approve
lvs delete entry <entry_id> --approve
```

``sh
lvs create po
lvs create po --pi <proforma_invoice>
```

```sh
lvs update stock <product_id> 
lvs update stock <product_id> --add/--remove
```

```sh



```
