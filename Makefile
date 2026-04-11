.PHONY: build test test-formula lint install clean

BINARY := beads-plan
BUILD_DIR := ./build
EXAMPLE_CHANGE := openspec/changes/example

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/beads-plan/

test: test-formula
	go test ./... -v

# test-formula runs shell-based smoke tests on the meow-openspec
# formula: seed, required-var enforcement, and a full cook round-trip
# that verifies the step and gate counts. Runs against the example
# fixture at $(EXAMPLE_CHANGE).
test-formula:
	@echo "== formula seed =="
	bd mol seed meow-openspec --var change_dir=$(EXAMPLE_CHANGE) --var change=example
	@echo "== formula rejects missing required var =="
	@bd cook meow-openspec --mode runtime 2>&1 | grep -q "Missing: change_dir" \
		|| (echo "FAIL: expected 'Missing: change_dir' error when change_dir is omitted"; exit 1)
	@echo "ok"
	@echo "== formula cooks to 14 steps + 6 human gates =="
	@bd cook meow-openspec --mode runtime --var change_dir=$(EXAMPLE_CHANGE) --var change=example | \
		python3 -c 'import sys,json; d=json.load(sys.stdin); \
steps=d["steps"]; gates=[s["id"] for s in steps if s.get("gate",{}).get("type")=="human"]; \
assert len(steps)==14, f"expected 14 steps, got {len(steps)}"; \
assert len(gates)==6, f"expected 6 gates, got {len(gates)}: {gates}"; \
want=["proposal-review","specs-review","design-review","tasks-review","verify-review","archive-review"]; \
assert gates==want, f"gate order mismatch: want {want}, got {gates}"; \
print(f"ok: {len(steps)} steps, {len(gates)} gates: {gates}")'

lint:
	golangci-lint run ./...

install:
	go install ./cmd/beads-plan/

clean:
	rm -rf $(BUILD_DIR)
