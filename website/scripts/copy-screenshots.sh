#!/bin/bash
# Copia screenshots do repo para o diretório estático do Hugo.
# Executado pelo workflow de deploy;
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WEBSITE_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(dirname "$WEBSITE_DIR")"

SRC="$REPO_ROOT/docs/screenshots"
DST="$WEBSITE_DIR/static/img/screenshots"

mkdir -p "$DST"
cp "$SRC"/*.png "$DST/"
echo "Screenshots copiados: $(ls "$DST" | wc -l) arquivos"
