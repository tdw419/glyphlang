#!/bin/bash
# glyph-standalone.sh
# Bootstrapped execution of GlyphLang programs

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
GLYPH_BIN="$DIR/glyph"

# Use the bootstrap/main.glyph to run the target file
$GLYPH_BIN exec "$DIR/bootstrap/main.glyph" run "$1"
