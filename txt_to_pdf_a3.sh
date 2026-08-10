#!/usr/bin/env bash

set -euo pipefail

if ! command -v enscript >/dev/null 2>&1; then
  echo "Error: enscript is required but not found." >&2
  exit 1
fi

if ! command -v ps2pdf >/dev/null 2>&1; then
  echo "Error: ps2pdf is required but not found." >&2
  exit 1
fi

# Usage:
#   ./txt_to_pdf_a3.sh                 # convert all .txt files in current directory
#   ./txt_to_pdf_a3.sh 'SK25CPQ*.txt'  # convert only matching files
#   ./txt_to_pdf_a3.sh '/path/*.txt'   # convert files from another path
pattern="${1:-*.txt}"
font="${TXT2PDF_FONT:-Courier8}"
margin="${TXT2PDF_MARGIN:-18}"

shopt -s nullglob
files=( $pattern )
shopt -u nullglob

if [[ ${#files[@]} -eq 0 ]]; then
  echo "No files found for pattern: $pattern" >&2
  exit 1
fi

converted=0
for file in "${files[@]}"; do
  [[ -f "$file" ]] || continue

  out="${file%.*}.pdf"
  tmp_ps="$(mktemp /tmp/txt2pdf_a3_XXXXXX.ps)"
  echo "Converting: $file -> $out"

  # A3 landscape + monospaced font keeps wide invoice tables inside page bounds.
  enscript \
    --quiet \
    --no-header \
    --media=A4 \
    --portrait \
    --margins="$margin:$margin:$margin:$margin" \
    --font="$font" \
    "$file" \
    -p "$tmp_ps"

  ps2pdf -sPAPERSIZE=a3 "$tmp_ps" "$out"
  rm -f "$tmp_ps"

  converted=$((converted + 1))
done

echo "Done. Converted $converted file(s) to A3 PDF."
