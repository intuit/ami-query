NAME       = ami-query
VERSION := $(shell grep '^const version' ../version.go | awk -F\" '{print $$2}')
RPM_TOPDIR = "$(shell pwd)/rpmbuild"

RPMBUILD_ARGS := \
	--define "_topdir $(RPM_TOPDIR)" \
	--define "name $(NAME)" \
	--define "version $(VERSION)"

.PHONY: all
all: clean
	@mkdir -p $(RPM_TOPDIR)/{BUILD,RPMS,SRPMS}
	@mkdir -p $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)
	@cp -p ./rhel/{settings,ami-query.service} $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)/
	@cp -p ../{bin/ami-query,README.md} $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION)/
	@tar czf $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION).tar.gz -C $(RPM_TOPDIR)/SOURCES --exclude ".git" $(NAME)-$(VERSION)
	@rpmbuild $(RPMBUILD_ARGS) -ba --clean ./rhel/$(NAME).spec
	@cp -p $(RPM_TOPDIR)/RPMS/x86_64/$(NAME)-$(VERSION)*.x86_64.rpm ../

.PHONY: clean
clean:
	@rm -rf $(RPM_TOPDIR)

.PHONY: check-version
check-version:
ifndef VERSION
	$(error VERSION is undefined)
endif
