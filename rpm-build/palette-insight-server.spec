%define serviceuser insight
%define servicehome /etc/palette-insight-server


# Disable the stupid stuff rpm distros include in the build process by default:
#   Disable any prep shell actions. replace them with simply 'true'
%define __spec_prep_post true
%define __spec_prep_pre true
#   Disable any build shell actions. replace them with simply 'true'
%define __spec_build_post true
%define __spec_build_pre true
#   Disable any install shell actions. replace them with simply 'true'
%define __spec_install_post true
%define __spec_install_pre true
#   Disable any clean shell actions. replace them with simply 'true'
%define __spec_clean_post true
%define __spec_clean_pre true
# Disable checking for unpackaged files ?
#%undefine __check_files

# Use md5 file digest method.
# The first macro is the one used in RPM v4.9.1.1
%define _binary_filedigest_algorithm 1
# This is the macro I find on OSX when Homebrew provides rpmbuild (rpm v5.4.14)
%define _build_binary_file_digest_algo 1

# Use bzip2 payload compression
%define _binary_payload w9.bzdio


Name: palette-insight-server
Version: %version
Epoch: 400
Release: %buildrelease
Summary: Palette Insight Server
AutoReqProv: no
# Seems specifying BuildRoot is required on older rpmbuild (like on CentOS 5)
# fpm passes '--define buildroot ...' on the commandline, so just reuse that.
#BuildRoot: %buildroot
# Add prefix, must not end with /

Prefix: /

Group: default
License: Proprietary
Vendor: Palette Software
URL: http://www.palette-software.com
Packager: Palette Developers <developers@palette-software.com>

Requires: nginx

# Travis CI will make sure that we depend on the latest version
Requires: palette-insight-agent
Requires: palette-insight-toolkit
Requires: palette-supervisor

# Add the user for the service & setup SELinux
# ============================================

%pre
# Override the SELinux flag that disallows httpd to connect to the go process
# https://stackoverflow.com/questions/23948527/13-permission-denied-while-connecting-to-upstreamnginx
setsebool httpd_can_network_connect on -P

%post
# Start nginx on server start
/sbin/chkconfig nginx on

%postun
# TODO: we should switch back the httpd_can_network_connect flag for SELinux, IF we know that its safe to do so


# Generic RPM parts
# =================

%description
Palette Insight Server

%prep
# noop

%build
# noop

%install
# Create the logfile directory for supervisord
mkdir -p %{buildroot}/var/log/palette-insight-server/

# Create the storage directory under /data
mkdir -p %{buildroot}/data/insight-server/licenses

%clean
# noop

%files
%defattr(-,root,root,-)

# Reject config files already listed or parent directories, then prefix files
# with "/", then make sure paths with spaces are quoted. I hate rpm so much.
/usr/local/bin/palette-insight-server

%attr(-,insight, insight) %dir /data/insight-server/licenses
%attr(-,insight, insight) %dir /data/insight-server
%dir /var/log/palette-insight-server

%config /etc/palette-insight-server/server.config
%config /etc/nginx/conf.d/palette-insight-server.conf
%config /etc/supervisord.d/palette-insight-server.ini

%changelog
