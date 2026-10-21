PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1

.PHONY: build test install uninstall
build:
	go build -o why ./cmd/why

test:
	go test ./...

install: build
	install -d "$(DESTDIR)$(BINDIR)" "$(DESTDIR)$(MANDIR)"
	install -m 755 why "$(DESTDIR)$(BINDIR)/why"
	install -m 644 why.1 "$(DESTDIR)$(MANDIR)/why.1"

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/why" "$(DESTDIR)$(MANDIR)/why.1"
