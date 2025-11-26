# Define the binary path and version info
BINARY = bpfva
BUILD_DIR = ./bin

.PHONY: build test compile-tests validate clean

# 1. BUILD: Creates the bin directory and outputs the executable to ./bin/bpfva
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd

# 2. COMPILE TESTS: Compiles all C files into .o files (Required before validation)
compile-tests:
	@mkdir -p test/compiled
	@echo "Compiling C test programs..."
	# Compile test/programs/*.c files
	@find test/programs -name "*.c" | while read file; do \
		echo "  Compiling $$file..."; \
		clang -O2 -target bpf -c $$file -o test/compiled/$$(basename $$file .c).o || exit 1; \
	done
	# Compile test/validation/*.c files
	@find test/validation -name "*.c" | while read file; do \
		echo "  Compiling $$file..."; \
		clang -O2 -target bpf -c $$file -o test/compiled/$$(basename $$file .c).o || exit 1; \
	done
	@echo "Done compiling test programs."

# 3. VALIDATE: Runs build, compilation, and the validation script.
validate: build compile-tests
	@echo "Running validation tests (requires sudo)..."
	@sudo -E bash scripts/validate.sh

# 4. CLEAN: Removes the build directory and all generated test files.
clean:
	rm -rf $(BUILD_DIR)
	rm -rf test/compiled/*.o
	rm -f validation_results.txt