.PHONY: all

# TODO: build monit from source
# https://bitbucket.org/tildeslash/monit/

BINARIES := $(ROOTFS)/lib/monit/init/monitrc

all: monit.mk $(BINARIES)

$(BINARIES):
	@echo "\\n---> Building monit\\n"
	cp $(ASSETS)/monit/monit $(TARGETDIR)/
	cp -af $(ASSETS)/makefiles/master/monit/monit.service $(ROOTFS)/lib/systemd/system/
	ln -sf /lib/systemd/system/monit.service  $(ROOTFS)/lib/systemd/system/multi-user.target.wants/
	install -m 0755 $(TARGETDIR)/monit $(ROOTFS)/usr/bin
	mkdir -p $(ROOTFS)/lib/monit/init
	mkdir -p $(ROOTFS)/etc/monit
	cp ./*.conf $(ROOTFS)/lib/monit/init
	install -m 0755 ./monitrc $(ROOTFS)/lib/monit/init/
