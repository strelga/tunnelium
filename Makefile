BINARY = tunnelium
VERSION_FILE = VERSION

# Current version is read from the VERSION file (e.g.: 0.1.0)
VERSION := $(shell cat $(VERSION_FILE) 2>/dev/null || echo "0.0.0")

.PHONY: build build-linux build-darwin build-darwin-arm64 release \
        test clean publish patch minor major

build:
	go build -o $(BINARY) ./cmd/tunnelium

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(BINARY)-linux-amd64 ./cmd/tunnelium

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(BINARY)-darwin-amd64 ./cmd/tunnelium

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $(BINARY)-darwin-arm64 ./cmd/tunnelium

release: build-linux build-darwin build-darwin-arm64

test:
	go test ./...

clean:
	rm -f $(BINARY) $(BINARY)-*

# --- Releases ---

# Publish the current version to GitHub via gh CLI
publish: release
	gh release create v$(VERSION) \
		$(BINARY)-linux-amd64 \
		$(BINARY)-darwin-amd64 \
		$(BINARY)-darwin-arm64 \
		--title "v$(VERSION)" \
		--generate-notes

# Common bump-version template: computes NEW_VERSION, commits, tags, and pushes
define bump_version
	$(eval NEW_VERSION := $(shell echo "$(VERSION)" | awk -F. $(1)))
	echo "$(NEW_VERSION)" > $(VERSION_FILE)
	git add $(VERSION_FILE)
	git commit -m "release: v$(NEW_VERSION)"
	git tag v$(NEW_VERSION)
	git push origin main v$(NEW_VERSION)
	@echo "Released v$(NEW_VERSION) — CI will build and publish binaries"
endef

patch:
	$(call bump_version,'{printf "%d.%d.%d", $$1, $$2, $$3+1}')

minor:
	$(call bump_version,'{printf "%d.%d.0", $$1, $$2+1}')

major:
	$(call bump_version,'{printf "%d.0.0", $$1+1}')
