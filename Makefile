TAGS := webkit2_41

dev:
	wails dev -tags $(TAGS)

build:
	wails build -tags $(TAGS)

.PHONY: dev build
