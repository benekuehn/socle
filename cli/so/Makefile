# Makefile for the Socle project (located in cli/so)

# Use Go to generate the CLI reference documentation in cli/so/README.md
docs:
	@echo "Generating CLI documentation..."
	@go run ./internal/docgen/main.go
	@echo "CLI documentation updated in cli/so/README.md"

.PHONY: docs 