Name: nicety
Version: 0.1.0
Release: 1%{?dist}
Summary: Process priority management daemon

License: MIT
URL: https://github.com/jcrd/nicety
Source0: https://github.com/jcrd/nicety/archive/v0.1.0.tar.gz

Requires: extrace
Requires: util-linux

BuildRequires: go
BuildRequires: perl

%global debug_package %{nil}

%description
nicety manages the priority of processes based on udev-like rules.

%prep
%setup

%build
%make_build PREFIX=/usr

%install
%make_install PREFIX=/usr
mkdir -p $RPM_BUILD_ROOT/%{_sysconfdir}/%{name}/rules.d

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_sysconfdir}/%{name}/rules.d
/usr/lib/systemd/system/%{name}.service
%{_mandir}/man1/%{name}.1.gz

%changelog
* Mon Jan 25 2021 James Reed <jcrd@tuta.io> - 0.1.0-1
- Initial package
