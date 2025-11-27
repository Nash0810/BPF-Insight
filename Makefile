# Define the binary path and version info
BINARY = bpfva
BUILD_DIR = ./bin

.PHONY: build test compile-tests validate clean install visualize-all

# 1. BUILD: Creates the bin directory and outputs the executable to ./bin/bpfva
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd

# 2. TEST: Run unit tests
test:
	go test ./... -v

# 3. COMPILE TESTS: Compiles all C files into .o files (Required before validation)
compile-tests:
	@mkdir -p test/compiled
	@echo "Compiling C test programs..."
	# Compile test/programs/*.c files
	@find test/programs -name "*.c" 2>/dev/null | while read file; do \
		echo "  Compiling $$file..."; \
		clang -O2 -target bpf -c $$file -o test/compiled/$$(basename $$file .c).o || exit 1; \
	done || true
	# Compile test/validation/*.c files
	@find test/validation -name "*.c" 2>/dev/null | while read file; do \
		echo "  Compiling $$file..."; \
		clang -O2 -target bpf -c $$file -o test/compiled/$$(basename $$file .c).o || exit 1; \
	done || true
	@echo "Done compiling test programs."

# 4. VALIDATE: Runs build, compilation, and the validation script.
validate: build compile-tests
	@echo "Running validation tests (requires sudo)..."
	@sudo -E bash scripts/validate.sh

# 5. VISUALIZE-ALL: Generate CFG visualizations for all test programs
visualize-all: build compile-tests
	@mkdir -p examples
	@for f in test/compiled/*.o; do \
		if [ -f "$$f" ]; then \
			echo "Visualizing $$(basename $$f)..."; \
			$(BUILD_DIR)/$(BINARY) visualize $$f --render || true; \
		fi \
	done

# 6. INSTALL: Install binary to /usr/local/bin
install: build
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/

# 7. CLEAN: Removes the build directory and all generated test files.
clean:
	rm -rf $(BUILD_DIR)
	rm -rf test/compiled/*.o
	rm -f validation_results.txt
	rm -rf examples/*.dot examples/*.png