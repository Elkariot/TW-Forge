TAGS := webkit2_41

dev:
	WEBKIT_DISABLE_DMABUF_RENDERER=1 wails dev -tags $(TAGS)

build:
	wails build -tags $(TAGS)

.PHONY: dev build
