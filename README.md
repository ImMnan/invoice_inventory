

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
lvs apply -f <file_path> 
lvs apply -f <file_path> --approve 
```

```sh
lvs delete invoice <invoice_id> --approve
lvs delete entry <entry_id> --approve
```

``sh
lvs add po
lvs add po --pi <proforma_invoice>
```

```sh
lvs update stock <product_id> 
lvs update stock <product_id> --add/--remove
```


```sh
```
