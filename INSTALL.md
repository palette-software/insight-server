# Installing the palette insight server

## Pre-requisites

The palette insight server package is designed to be installed on RedHat
and CentOS linux boxes.

The only extra requirement for the server is the EPEL package repository
for ```supervisord``` and ```nginx```. To add it, on RH 7, use the
following:

```bash
wget https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
sudo rpm -Uvh epel-release-latest-7.noarch.rpm
```

## Getting the RPM packages for the Insight Server installation

There are two packages used for installing the server:


```
palette-insight-certs-1.0.0-1.noarch.rpm
```

which contains the HTTPS certificates to use (it does not change often,
and requires more delicate handling then the server package), and

```
palette-insight-server-$VERSION-1.x86_64.rpm
```

which has the server, Nginx and Supervisord as dependencies and some
configuration files to run it the following way:

* Nginx serves port 443 with the certificates from ```palette-insight-certs```
* It forwards the https requests to port 9443 where the service is
  listening.
* The service itself is ran through supervisord which should restart it
  on failiures and should handle the logrotation.

Both of these RPMs can be downloaded from the [github releases of
insight-server](https://github.com/palette-software/insight-server/releases).


## Installing the service


So to install the service from these two RPMs:

```bash
sudo yum install -y palette-insight-server-v1.3.6-1.x86_64.rpm palette-insight-certs-1.0.0-1.noarch.rpm
```


This will install all nginx, supervisord and the insight server, and
provides a basic configuration.


## Configuring the service

First determine where you want to put the data:

```bash
[ec2-user@ip-172-31-17-158 ~]$ df -h
Filesystem      Size  Used Avail Use% Mounted on
/dev/xvda2       10G  1.9G  8.1G  19% /
devtmpfs         16G     0   16G   0% /dev
tmpfs            16G     0   16G   0% /dev/shm
tmpfs            16G   17M   16G   1% /run
tmpfs            16G     0   16G   0% /sys/fs/cgroup
/dev/xvdb1      4.9T  786M  4.9T   1% /data
tmpfs           3.2G     0  3.2G   0% /run/user/1000
tmpfs           3.2G     0  3.2G   0% /run/user/1001
```

In this case, we'll put the data into ```/data/insight-server```

To do this, we have to edit the server configuration file, which is by
default available at:

```
/etc/palette-insight-server/server.config
```

The default configuration file puts all data under ```/tmp``` so we have
to edit it:

```
# PATHS
# =====

# The root directory for the uploads to go into.
upload_path=/tmp/insight-server/uploads

# The path where the maxid files are stored
maxid_path=/tmp/insight-server/maxids

# The path where the licenses are stored
licenses_path=/tmp/insight-server/licenses

# The directory where the update files for the agent are stored.
updates_path=/tmp/insight-server/updates

# SERVER
# ======

# The address to bind to. Leave empty for default which is 0.0.0.0
bind_address=

# The port the server is binding itself to
port=9443

# SSL
# ===

# As we are using Nginx to forward the HTTPS requests to our port, we
# generally dont need to run with TLS

# Should the server use SSL?
#tls=true

# The locations of the SSL certificate and key files
#cert=/tmp/insight-server/ssl-certs/star_palette-software_net.crt
#key=/tmp/insight-server/ssl-certs/server.key
```


After we change the paths (and remove the unused TLS part, which we do
not need as we are served by nginx):

```
# PATHS
# =====

# The root directory for the uploads to go into.
upload_path=/data/insight-server/uploads

# The path where the maxid files are stored
maxid_path=/data/insight-server/maxids

# The path where the licenses are stored
licenses_path=/data/insight-server/licenses

# The directory where the update files for the agent are stored.
updates_path=/data/insight-server/updates

# SERVER
# ======

# The address to bind to. Leave empty for default which is 0.0.0.0
bind_address=

# The port the server is binding itself to
port=9443

```

Now we need to create the directory where the uploads will be stored
(the service should automatically create these, but it does not have
permission by default to create anything in ```/data```)

```
sudo mkdir -p /data/insight-server/licenses
sudo chown -R insight:insight /data/insight-server
```


## Adding a license

To add a license for a tenant on this server, add a file with a ```.license```
extension to the licenses path of the server (in our example:

```/data/insight-server/licenses```

).

On boot the server should log all licenses it manages to load.

## Starting the server

### Starting supervisord

By default, ```supervisord``` is responsible for running and restarting
the server and handling its log rotation.

To start supervisord after the install, simply:

```
sudo service supervisord start
```

After this completes (or if we made any changes to the servers
supervisor setup), we need to reload the services list:

```
sudo supervisorctl reload
```

### Checking server status

This should start the server. The status of the server can be checked:

```
sudo supervisorctl status
```

When starting the server, supervisord waits for 10 seconds of clean
runtime before it considers a service running, so for the first 10
seconds, this status should be ```STARTING```

```
[ec2-user@ip-172-31-17-158 ~]$ sudo supervisorctl status
palette-insight-server           STARTING
```

After supervisor considers the service successfully started:

```
[ec2-user@ip-172-31-17-158 ~]$ sudo supervisorctl status
palette-insight-server           RUNNING   pid 23119, uptime 0:00:11
```

You can check if the server works by

```
curl http://localhost:9443/updates/new-version
```

This should return an HTML document.


### Starting nginx

Nginx provides the fronted for the server, and should already be
configured by the RPM package.

```
sudo service nginx start
```

And at this point checking

```
https://xxx-insight.palette-software.net/updates/new-version
```

should return the same html document and should have a valid
certificate.

# Upgrading the server

Simply run.

```
sudo yum install palette-insight-server-NEW_VERSION.rpm
```

All configuration files should be left alone.


# Removing the server

```bash
sudo yum remove -y palette-insight-server palette-insight-certs nginx supervisor
```


# File locations

- server executable: ```/usr/local/bin/palette-insight-server```
- server config: ```/etc/palette-insight-server/server.config```
- server log files: ```/var/log/palette-insight-server```

- supervisor config: ```/etc/supervisord.d/palette-insight-server.ini```
- nginx site: ```/etc/nginx/conf.d/palette-insight-server.conf```

- certificates: ```/etc/palette-insight-certs```
