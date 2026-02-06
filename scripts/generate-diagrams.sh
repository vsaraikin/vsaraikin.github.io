#!/bin/bash
# Generate SVG diagrams from Mermaid files across all page bundles
#
# Usage: ./generate-diagrams.sh [file.mmd]
#   If no file specified, generates all .mmd files in content/posts/

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

generate_svg() {
    local mmd_file="$1"
    local svg_file="${mmd_file%.mmd}.svg"

    echo "Generating: $mmd_file → $svg_file"

    if curl -s -X POST https://kroki.io/mermaid/svg \
        --data-binary "@$mmd_file" \
        -o "$svg_file"; then

        if head -1 "$svg_file" | grep -q "^<svg"; then
            echo "  ✓ Success ($(wc -c < "$svg_file" | tr -d ' ') bytes)"
        else
            echo "  ✗ Error: $(cat "$svg_file")"
            rm "$svg_file"
            return 1
        fi
    else
        echo "  ✗ Failed to reach kroki.io"
        return 1
    fi
}

if [ -n "$1" ]; then
    generate_svg "$1"
else
    count=0
    while IFS= read -r -d '' mmd_file; do
        generate_svg "$mmd_file"
        ((count++))
    done < <(find "$SCRIPT_DIR/content/posts" -name "*.mmd" -print0)
    echo ""
    echo "Generated $count diagram(s)"
fi
