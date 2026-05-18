########################################
### Simulations
#
# Three CI-friendly gates around the SDK v0.54 simsx runner:
#   - simple             single seed × 500 blocks, fast smoke
#   - nondeterminism     3 seeds × 100 blocks × 3 runs each, state-hash parity
#   - multi-seed-short   3 seeds × 50 blocks × 3 runs each, broader determinism
#
# Heavy variants (multi-seed-long, benchmark, profile, fuzz) deliberately omitted
# — consumer chains exercise full sim batteries against their own apps; the
# x-stream module just needs to prove its own ops don't violate state invariants.

SIMAPP = ./simapp/...

SIM_NUM_BLOCKS ?= 500
SIM_BLOCK_SIZE ?= 50
SIM_COMMIT ?= true

test-sim-simple:
	@echo "Running simple single-seed simulation (NumBlocks=$(SIM_NUM_BLOCKS), BlockSize=$(SIM_BLOCK_SIZE))..."
	@go test -mod=readonly -tags=sims $(SIMAPP) -run TestFullAppSimulation -Enabled=true \
		-NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -Period=100 -v -timeout 24h

test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly -tags=sims $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

# Faster determinism smoke. Same TestAppStateDeterminism harness as the
# nondeterminism target but with shorter chain depth, suitable as a per-PR gate.
test-sim-multi-seed-short:
	@echo "Running short multi-seed determinism check."
	@go test -mod=readonly -tags=sims $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=50 -BlockSize=10 -Commit=true -Period=0 -v -timeout 24h

.PHONY: test-sim-simple test-sim-nondeterminism test-sim-multi-seed-short
