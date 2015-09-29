NAME       = ami-query
VERSION    = 1.0.0
RPM_TOPDIR = "$(shell pwd)/rpmbuild"

RPMBUILD_ARGS := \
	--define "_topdir $(RPM_TOPDIR)" \
	--define "name $(NAME)" \
	--define "version $(VERSION)"

.PHONY: all clean

all: clean
	@mkdir -p $(RPM_TOPDIR)/{BUILD,RPMS,SRPMS}
	@mkdir -p $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)
	@cp -p ./rhel/{settings,rc.ami-query,ami-query.service} $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)/
	@cp -p ../{bin/ami-query,README.md} $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)/
	@tar czf $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION).tar.gz -C $(RPM_TOPDIR)/SOURCES --exclude ".git" $(NAME)-$(VERSION)
	@rpmbuild $(RPMBUILD_ARGS) -ba --clean ./rhel/$(NAME).spec
	@cp -p $(RPM_TOPDIR)/RPMS/x86_64/$(NAME)-$(VERSION)*.x86_64.rpm ../

clean:
	@echo "==> Cleaning up previous RPMs builds."
	@rm -rf $(RPM_TOPDIR)
