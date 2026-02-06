.PHONY: serve diagrams build clean

# Run Hugo dev server with drafts
serve: diagrams
	hugo server --buildDrafts --disableFastRender

# Generate SVG diagrams from all .mmd files
diagrams:
	@./scripts/generate-diagrams.sh

# Build production site
build: diagrams
	hugo --minify

# Build with drafts
build-drafts: diagrams
	hugo --buildDrafts --minify

# Clean generated files
clean:
	rm -rf public/ resources/_gen/
