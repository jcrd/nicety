VERSIONCMD = git describe --dirty --tags --always 2> /dev/null
VERSION := $(shell $(VERSIONCMD) || cat VERSION)

PREFIX ?= /usr/local
BINPREFIX ?= $(PREFIX)/bin
LIBPREFIX ?= $(PREFIX)/lib
MANPREFIX ?= $(PREFIX)/share/man

MANPAGE = nicety.1

all: nicety $(MANPAGE)

nicety: nicety.go
	go build -ldflags="-X 'main.version=$(VERSION)'" $<

$(MANPAGE): man/$(MANPAGE).pod
	pod2man -n=nicety -c=nicety -r=$(VERSION) $< $(MANPAGE)

install:
	mkdir -p $(DESTDIR)$(BINPREFIX)
	cp -p nicety $(DESTDIR)$(BINPREFIX)
	mkdir -p $(DESTDIR)$(LIBPREFIX)/systemd/system
	cp -p systemd/nicety.service $(DESTDIR)$(LIBPREFIX)/systemd/system
	mkdir -p $(DESTDIR)$(MANPREFIX)/man1
	cp -p $(MANPAGE) $(DESTDIR)$(MANPREFIX)/man1

uninstall:
	rm -f $(DESTDIR)$(BINPREFIX)/nicety
	rm -f $(DESTDIR)$(LIBPREFIX)/systemd/system/nicety.service
	rm -f $(DESTDIR)$(MANPREFIX)/man1/$(MANPAGE)

clean:
	rm -f nicety $(MANPAGE)

.PHONY: all install uninstall clean
