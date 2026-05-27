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

patch minor major:
	@v=$$(cat $(VERSION_FILE)) && \
	major=$$(echo "$$v" | cut -d. -f1) && \
	minor=$$(echo "$$v" | cut -d. -f2) && \
	patch_num=$$(echo "$$v" | cut -d. -f3) && \
	case "$@" in \
	  patch) new="$$major.$$minor.$$((patch_num + 1))" ;; \
	  minor) new="$$major.$$((minor + 1)).0" ;; \
	  major) new="$$((major + 1)).0.0" ;; \
	esac && \
	echo "$$new" > $(VERSION_FILE) && \
	git add $(VERSION_FILE) && \
	git commit -m "release: v$$new" && \
	git tag v$$new && \
	git push origin main v$$new && \
	echo "Released v$$new — CI will build and publish binaries"
