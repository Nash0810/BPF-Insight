.PHONY: build test compile-tests clean

build:
	go build -o bpfva ./cmd

test:
	go test ./...

compile-tests:
	# Will compile C test programs (Iteration 1)
	@echo "Compiling test programs... (Iteration 1 not implemented)"

clean:
	rm -f bpfva
