TAGS := webkit2_41

# Linux/Mac dev (WebKit)
dev:
	WEBKIT_DISABLE_DMABUF_RENDERER=1 wails dev -tags $(TAGS)

# Linux/Mac build
build:
	wails build -tags $(TAGS)

# Windows dev (PowerShell: make dev-win)
dev-win:
	wails dev

# Windows build (PowerShell: make build-win)
build-win:
	wails build

.PHONY: dev build dev-win build-win
