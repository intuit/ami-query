Name:    %{name}
Version: %{version}
Release: 1%{?dist}
Summary: A RESTful API service to query Amazon AWS AMIs.
URL:     https://github.com/intuit/ami-query
License: MIT
Group:   Application/SystemTools
Source0: %{name}-%{version}.tar.gz

BuildRoot: %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)
BuildArch: x86_64

%define debug_package %{nil}
%define _unpackaged_files_terminate_build 0

%description
Provide a RESTful API to query information about Amazon AWS AMIs.

%pre
if grep ^ami-query: /etc/group >> /dev/null ; then
  : # group already exists
else
  groupadd ami-query
fi

if grep ^ami-query: /etc/passwd >> /dev/null ; then
  : # user already exists
else
  useradd -g ami-query -s /sbin/nologin ami-query
fi

%prep
%setup -q

%build

%install
rm -rf $RPM_BUILD_ROOT
install -m 0755 -d $RPM_BUILD_ROOT/usr/bin/
install -m 0755 ami-query $RPM_BUILD_ROOT/usr/bin/ami-query
install -m 0755 -d $RPM_BUILD_ROOT/etc/sysconfig/
install -m 0640 settings $RPM_BUILD_ROOT/etc/sysconfig/ami-query

install -m 0755 -d $RPM_BUILD_ROOT/usr/lib/systemd/system/
install -m 0644 ami-query.service $RPM_BUILD_ROOT/usr/lib/systemd/system/ami-query.service

%clean
rm -rf $RPM_BUILD_ROOT

%files
%defattr(-,root,root,-)
%doc README.md
/usr/bin/ami-query
%config(noreplace) %attr(0640,root,ami-query) /etc/sysconfig/ami-query

/usr/lib/systemd/system/ami-query.service

%changelog
* Thu Sep 24 2015 James Massara <james_massara@intuit.com>
See https://github.com/intuit/ami-query/blob/master/CHANGELOG.md
