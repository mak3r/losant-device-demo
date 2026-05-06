BINARY     := ldc-demo
MODULE     := github.com/mak3r/ldc-demo
BUILD_DIR  := bin
INSTALL_DIR := /usr/local/bin

.PHONY: build test lint install clean tofu-fmt tofu-validate tidy

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/ldc-demo

test:
	go test ./...

lint:
	golangci-lint run ./...

install: build
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

tofu-fmt:
	tofu fmt -recursive tofu/

tofu-validate:
	@for dir in tofu/modules/*/; do \
		echo "Validating $$dir ..."; \
		tofu -chdir="$$dir" init -backend=false -input=false > /dev/null && \
		tofu -chdir="$$dir" validate; \
	done

tidy:
	go mod tidy
