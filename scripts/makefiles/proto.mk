###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.18.1
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace --user $(shell id -u):$(shell id -g) $(protoImageName)

# protoReclaimOwnership: under rootless docker, files written by the protoc/buf
# container come back owned by an unprivileged sub-UID on the host. Run a root
# container with the same bind-mount and chown everything back to whatever uid
# the host sees as the workspace root. Under rootful docker this is a no-op.
define protoReclaimOwnership
@$(DOCKER) run --rm -v $(CURDIR):/workspace --user 0:0 $(protoImageName) sh -c \
	'OWN=$$(stat -c "%u:%g" /workspace); chown -R "$$OWN" /workspace/proto /workspace/api /workspace/github.com 2>/dev/null; true'
endef

proto-all: proto-format proto-lint proto-gen proto-pulsar-gen

proto-gen:
	@echo "Generating Protobuf files"
	@chmod 777 proto && chmod 666 proto/buf.lock
	@mkdir -p github.com && chmod 777 github.com
	@$(protoImage) sh ./scripts/protocgen.sh
	$(protoReclaimOwnership)
	@cp -r github.com/unification-com/x-stream/* ./
	@rm -rf github.com
	@chmod 755 proto && chmod 644 proto/buf.lock

proto-pulsar-gen:
	@echo "Generating Protobuf Pulsar files"
	@chmod 777 proto && chmod 666 proto/buf.lock
	@chmod -R 777 api
	@$(protoImage) sh ./scripts/protocgen-pulsar.sh
	$(protoReclaimOwnership)
	@chmod 755 proto && chmod 644 proto/buf.lock
	@find api -type f -exec chmod 644 {} \; && find api -type d -exec chmod 755 {} \;

proto-format:
	@chmod 777 -R proto
	@$(protoImage) find ./proto -name "*.proto" -exec clang-format -i {} \;
	$(protoReclaimOwnership)
	@find proto -type f \( -name "*.proto" -o -name "*.yaml" -o -name "*.lock" \) -exec chmod 644 {} \;
	@find proto -type d -exec chmod 755 {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

proto-update-deps:
	@echo "Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-pulsar-gen proto-format proto-lint proto-check-breaking proto-update-deps
