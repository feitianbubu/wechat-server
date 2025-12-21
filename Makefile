.PHONY: image help

# Build Docker image
image:
	@echo "Building Docker image $(IMAGE_NAME):$(VERSION)..."
	@if [ ! -f VERSION ]; then \
		echo "$(VERSION)" > VERSION; \
	fi
	docker build -t $(IMAGE_NAME):$(VERSION) .
	docker tag $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest
	@echo "Successfully built image $(IMAGE_NAME):$(VERSION)"
	@echo "Also tagged as $(IMAGE_NAME):latest"