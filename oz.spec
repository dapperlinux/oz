Summary:    Sandbox system for workstation applications
Name:       oz
Version:    1
Release:    8

Group:      System Environment/Base
License:    BSD-3-Clause
Url:        https://github.com/subgraph/oz
Source0:    %{name}-%{version}.tar.gz
Source1:    oz-daemon.service
BuildArch:  x86_64

Requires: xpra
Requires: bridge-utils
Requires: ebtables
Requires: libacl
BuildRequires: go
BuildRequires: libacl-devel
BuildRequires: git



%description
Oz is a sandboxing system targeting everyday workstation applications. It acts as a wrapper around application executables for completely transparent user operations. It achieves process containment through the use of Linux Namespaces, Seccomp filters, Capabilities, and X11 restriction using Xpra. It has built-in support with automatic configuration of bridge mode networking and also support working with contained network environment using the built in connection forwarding proxy.

%prep
%autosetup

%build
#Setup GOPATH
mkdir -p %{_builddir}/gocode/src/github.com/subgraph/
mv oz %{_builddir}/gocode/src/github.com/subgraph/
export GOPATH=%{_builddir}/gocode

# Build GOdep
go get github.com/tools/godep

# Start the build
cd $GOPATH/src/github.com/subgraph/oz/
$GOPATH/bin/godep go install ./...


%install
export GOPATH=%{_builddir}/gocode

# Install the binaries
mkdir -p %{buildroot}%{_bindir}
cp $GOPATH/bin/oz* %{buildroot}%{_bindir}

# Install conf files
mkdir -p %{buildroot}%{_sysconfdir}/logrotate.d
mkdir -p %{buildroot}%{_sysconfdir}/network/if-up.d/
mkdir -p %{buildroot}%{_sysconfdir}/network/if-post-down.d/
mkdir -p %{buildroot}%{_sysconfdir}/NetworkManager/conf.d/
mkdir -p %{buildroot}%{_sysconfdir}/oz
mkdir -p %{buildroot}%{_sysconfdir}/rsyslog.d
mkdir -p %{buildroot}%{_sysconfdir}/sysctl.d
mkdir -p %{buildroot}%{_sysconfdir}/X11
mkdir -p %{buildroot}%{_sysconfdir}/xpra
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/logrotate.d/* %{buildroot}%{_sysconfdir}/logrotate.d/
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/network/if-up.d/* %{buildroot}%{_sysconfdir}/network/if-up.d/
chmod a+x %{buildroot}/etc/network/if-up.d/oz
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/NetworkManager/conf.d/oz.conf %{buildroot}%{_sysconfdir}/NetworkManager/conf.d/oz.conf
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/oz/* %{buildroot}%{_sysconfdir}/oz/
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/rsyslog.d/* %{buildroot}%{_sysconfdir}/rsyslog.d
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/sysctl.d/* %{buildroot}%{_sysconfdir}/sysctl.d
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/X11/* %{buildroot}%{_sysconfdir}/X11
cp $GOPATH/src/github.com/subgraph/oz/sources/etc/xpra/xpra.oz.conf %{buildroot}%{_sysconfdir}/xpra/xpra.oz.conf

# Install the service
mkdir -p %{buildroot}/lib/systemd/system
cp %{SOURCE1} %{buildroot}/lib/systemd/system/oz-daemon.service

# Make sym links
ln -s /etc/network/if-up.d/oz %{buildroot}/etc/network/if-post-down.d/oz

# Create Directories
mkdir -p %{buildroot}%{_prefix}/bin-oz
mkdir -p %{buildroot}/run/resolvconf
mkdir -p %{buildroot}%{_prefix}/lib/gvfs

%clean

%pre

%post
# Setup bridge networking
for INTERFACE in /sys/class/net/* ; do 
    INTERFACE="${INTERFACE%/}"; 
    INTERFACE="${INTERFACE##*/}"; 
    iptables -t nat -A POSTROUTING -o $INTERFACE -j MASQUERADE ; 
done
ebtables -P FORWARD DROP
ebtables -F FORWARD
ebtables -A FORWARD -i oz0 -j ACCEPT
ebtables -A FORWARD -o oz0 -j ACCEPT

# Start the sandbox service
systemctl enable oz-daemon.service
systemctl start oz-daemon.service

%files
# Binaries
%{_bindir}/oz*

# Conf files
%{_sysconfdir}/logrotate.d/oz-daemon
%{_sysconfdir}/network/if-up.d/oz
%{_sysconfdir}/network/if-post-down.d/oz
%{_sysconfdir}/NetworkManager/conf.d/oz.conf
%{_sysconfdir}/oz/*
%{_sysconfdir}/rsyslog.d/oz-daemon.conf
%{_sysconfdir}/sysctl.d/11-grsec-oz.conf
%{_sysconfdir}/sysctl.d/15-oz-net.conf
%{_sysconfdir}/X11/Xwrapper.config.oz
%{_sysconfdir}/xpra/xpra.oz.conf

# Services
/lib/systemd/system/oz-daemon.service

# Necessary Directories
%dir %{_prefix}/bin-oz
%dir /run/resolvconf
%dir %{_prefix}/lib/gvfs


%changelog
* Sun Jan 29 2017 Matthew Ruffell <msr50@uclive.ac.nz>
- Adding directories to start oz without errors

* Sun Dec  4 2016 Matthew Ruffell <msr50@uclive.ac.nz>
- First packaging
