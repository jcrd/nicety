Name: {{{ git_name name="nicety" }}}
Version: {{{ git_version lead="$(git tag | sed -n 's/^v//p' | sort --version-sort -r | head -n1)" }}}
Release: 1%{?dist}
Summary: Process priority management daemon

License: MIT
URL: https://github.com/jcrd/nicety
VCS: {{{ git_vcs }}}
Source0: {{{ git_pack }}}

Requires: extrace
Requires: util-linux

BuildRequires: go
BuildRequires: perl

%global debug_package %{nil}

%description
nicety manages the priority of processes based on udev-like rules.

%prep
{{{ git_setup_macro }}}

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
{{{ git_changelog }}}
