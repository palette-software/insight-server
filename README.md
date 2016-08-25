[![Build Status](https://travis-ci.com/palette-software/insight-server.svg?token=qWG5FJDvsjLrsJpXgxSJ&branch=master)](https://travis-ci.com/palette-software/insight-server)

# Palette Insight Webservice

## Starting the webservice

```bash
go get
go build && ./insight-server
```

or on Windows:

```
go build
insight-server.exe
```

## Configuration

The webservice uses the 'flag' library to handle configuration via both configuration files and environment variables.

For more information on the flag library see [github.com/namsral/flag](https://github.com/namsral/flag).

The environment variables and their corresponding configuration file values and command line flags are:

| Type   | Flag                                       | Environment                               | File                                      |
|--------|--------------------------------------------|-------------------------------------------|-------------------------------------------|
| string | -upload_path=/opt/insight-agent/uploads    | UPLOAD_PATH=/opt/insight-agent/uploads    | upload_path=/opt/insight-agent/uploads    |
| string | -maxid_path=/opt/insight-agent/maxids      | MAXID_PATH=/opt/insight-agent/maxids      | maxid_path=/opt/insight-agent/maxids      |
| string | -licenses_path=/opt/insight-agent/licenses | LICENSES_PATH=/opt/insight-agent/licenses | licenses_path=/opt/insight-agent/licenses |
| string | -updates_path=/opt/insight-agent/updates   | UPDATES_PATH=/opt/insight-agent/updates   | updates_path=/opt/insight-agent/updates
| string | -config dev.config                         | CONFIG=dev.config                         | config=dev.config                         |
| int    | -bind_port 8080                            | BIND_PORT=8080                            | bind_port=8080                            |
| string | -bind_address 127.0.0.1                    | BIND_ADDRESS=127.0.0.1                    | bind_address=127.0.0.1                    |
| bool   | -tls                                       | TLS=true                                  | tls=true                                  |
| string | -cert certs/cert.pem                       | CERT=certs/cert.pem                       | cert=certs/cert.pem                       |
| string | -key certs/key.pem                         | KEY=certs/key.pem                         | key=certs/key.pem                         |
| string | -logformat json                            | LOGFORMAT=text                            | logformat=color                           |
| string | -loglevel warn                             | LOGLEVEL=debug                            | loglevel=info                             |

To get a list of command line options, use the ```--help``` switch. On my machine (windows) this results in:

```
./server.exe --help
Usage of C:\Users\Miles\go\src\github.com\palette-software\insight-server\server\server.exe:
  -archive_path="": The directory where the uploaded serverlogs are archived.
  -bind_address="": The address to bind to. Leave empty for default .
  -cert="cert.pem": The TLS certificate file to use when tls is set.
  -config="": Configuration file to use.
  -key="key.pem": The TLS certificate key file to use when tls is set.
  -licenses_path="C:\\Users\\Miles\\go\\src\\github.com\\palette-software\\insight-server\\server\\licenses": The directory the licenses are loaded from on start.
  -logformat="text": The log format to use ('json' or 'text' or 'color')
  -loglevel="info": The log level to use ('info', 'warn' or 'debug')
  -maxid_path="C:\\Users\\Miles\\AppData\\Local\\Temp\\uploads\\maxid": The root directory for the maxid files to go into.
  -port=9000: The port the server is binding itself to
  -tls=false: Use TLS for serving through HTTPS.
  -updates_path="C:\\Users\\Miles\\go\\src\\github.com\\palette-software\\insight-server\\server\\updates": The directory where the update files for the agent are stored.
  -upload_path="C:\\Users\\Miles\\AppData\\Local\\Temp\\uploads": The root directory for the uploads to go into.
```

## Sample configuration file

A sample configuration file can be found in the ```server``` folder as ```sample.config```

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

# SSL
# ===

# As we are using Nginx to forward the HTTPS requests to our port, we
# generally dont need to run with TLS

# Should the server use SSL?
#tls=true

# The locations of the SSL certificate and key files
#cert=/data/insight-server/ssl-certs/star_palette-software_net.crt
#key=/data/insight-server/ssl-certs/server.key

# LOGGING
# =======

# Sets the minimal log level. Can be 'debug', 'info', 'warn', 'error'
loglevel=info

# Sets the output format for logs. Can be 'json', 'text', 'color' (the last one
# force color output for windows terminals
logformat=json
```

This configuration file gets installed as default when using the RPM installer.

## IpTables

To allow the service to listen to port 443 without sudo privileges an IpTables forwarding needs to be set up.

```bash
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 443 -j REDIRECT --to-port 9443
```

## User authentication

On startup, the server tries to load all licenses from the ```licenses_path``` directory (or the equivalent env/config option).

The usernames are the ```licenseId``` field of the license, the authentication token is the ```token``` field of the license.

## MaxIds

For streaming tables, the webservice provides an endpoint and upload integration:

* the agent sends a ```maxid``` field with the streaming table CSV files, which designates the last record sent by the agent from
  that table

```
POST /upload?pkg=public&table=http_requests&maxid=abcdef123
```

* later the agent can retrieve this ```maxid``` for the specific table by:

```
GET /maxid?table=http_requests
```
 
* ```maxid``` must be a string


The maxids are stored in the directory set by the ```maxid_path`` flag.

## Checking if the service is running

```
$ curl http://localhost:9000/
PONG
```

## Uploading a file

See the openAPI documentation inside the docs/generated folder

--

## Autoupdate service endpoints

The service provides support for sending updated installers to the agents. All updates are based on two
parts: the __PRODUCT__ (like ```agent```) and the __VERSION__ (like ```v1.3.2```).

### Adding a new version of a product

Navigate your browser to:

```
http://SERVER/updates/new-version
```

This should present an HTML form where you can select the product name and the new version and upload a new
file for this version.

### Getting the latest verion of a product

Send a GET request to:

```
GET http://SERVER/updates/latest-version?product=PRODUCT_NAME
 => 200: {"Major":1,"Minor":9,"Patch":3,"Product":"agent","Md5":"6a6d0cc56d7186ba54fccca2ae7fcda8","Url":"/updates/products/agent/v1.9.3/agent-v1.9.3"}
```

The JSON response contains the
* Major version
* Minor version
* Patch version
* The Md5 of the file
* The download path on the server (currently its only a path as the server address may be different for the agent and the server)


If the given product has no versions (most likely because of an invalid product name) then the server returns a 404 response:

```
GET http://localhost:9000/updates/latest-version?product=agenr
 => 404: Cannot find product 'agenr': Cannot find product 'agenr'
```


### Getting the update files

After the agent has the latest version information from the ```/uploads/latest-version?product=...``` endpoint, it can download
the file by issuing a request to the file path in the response:

```
GET http://localhost:9000/updates/products/agent/v1.9.3/agent-v1.9.3
 => 200 CONTENTS_OF_THE_FILE
```

--

## API Documentation

A basic documentation using OpenAPI is available in the docs folder, or
a HTML-ized version is available in the docs/generated folder.

## Assets

The server uses [go-bindata](https://github.com/jteeuwen/go-bindata) to package its assets into
a go source file so that the server itself has no dependencies on runtime data.

To install it:

```
go get -u github.com/jteeuwen/go-bindata/...
```

(dont forget the three dots from the end).

Later running

```
go generate -x github.com/palette-software/insight-server
```

should update the asset package used by the server for future builds. (The ```-x``` switch simply displays the commands
ran by ```go generate```).

Important note: please check in the generated sources into the git tree, because:


> There is one thing you need to be aware of when using go generate. The tool isn’t
> integrated with go get, as one might expect. Because of that, your project will only
> be “go gettable” if you check in all sources created by go generate.

# RPMs

## Install steps using the palette RPM repo

### Install

These steps assume that the data directory is ```/data``` (as recommended by the Palette Guidelines).

```bash
sudo yum install -y  wget

# Add the EPEL Repo (for Nginx and Supervisord)
wget http://dl.fedoraproject.org/pub/epel/7/x86_64/e/epel-release-7-5.noarch.rpm
sudo rpm -ivh epel-release-7-5.noarch.rpm


# Add the repo
sudo yum-config-manager --add-repo=https://rpm.palette-software.com/redhat/

# Now we need to disable GPG checks for this repo. Edit the repo file with:
sudo vi /etc/yum.repos.d/rpm.palette.software.com_redhat_.repo_

# Add this line to the end of the repo file (without the comment)
# gpgcheck=0


# Install the server + nginx + supervisord + certs
sudo yum install -y palette-insight-server


# Configure the setup
# -------------------

# Create the server directory and the license subdir, so we can put the license there
sudo mkdir -p /data/insight-server/licenses




# Update configuration with the correct paths
vim /etc/palette-insight-server/server.config

# Add the license
sudo vim /data/insight-server/licenses/<LICENSE NEV>.license

# Change the owner
sudo chown -R insight:insight /data/insight-server

# Start the supervisor & nginx
sudo service supervisord start
sudo service nginx start

# Start nginx on server start
sudo /sbin/chkconfig nginx on

# Start supervisord on server start
sudo /sbin/chkconfig supervisord on
```


### Update


```bash
# Get the server status
sudo supervisorctl status
# => palette-insight-server           RUNNING   pid 11799, uptime 0:04:05


# Update the server
sudo yum update palette-insight-server


# Restart supervisord
sudo supervisorctl restart palette-insight-server


# Check if its running correctly (wait 10 seconds)
sudo supervisorctl status
# => palette-insight-server           RUNNING   pid 11799, uptime 0:04:05

```

## Installing from rpms

The service requires two rpm-s to install:

```
palette-insight-certs-1.0.0-1.noarch.rpm
```

which contains the HTTPS certificates to use (it does not change often,
and requires more delicate handling then the server package), and

```
palette-insight-server-v1.8.2-1.x86_64.rpm
```

which has the server, Nginx and Supervisord as dependencies and some
configuration files to run it the following way:

* Nginx serves port 443 with the certificates from ```palette-insight-certs```
* It forwards the https requests to port 9443 where the service is
  listening.
* The service itself is ran through supervisord which should restart it
  on failiures and should handle the logrotation.

So to install the service from these two RPMs (add EPEL before as a repo):

```bash
sudo yum install -y ./palette-insight-server-v1.8.2-1.x86_64.rpm ./palette-insight-certs-1.0.0-1.noarch.rpm
```

and to remove:

```bash
sudo yum remove -y palette-insight-server palette-insight-certs nginx supervisor
```

## Building the RPMs

(A working CentOS/RedHat installation is recommended, but not required).

Building the rpms of course needs the ```rpm``` & ```rpm-build``` tools
on the build system.

To build the certificates package, you need to download the certificates
you wish to include and extract them to the

```
rpm-build/etc/palette-insight-certs
```

folder as ```cert.crt``` and ```cert.key```


After the server has been built with ```go build``` you can build the
rpms with:

```bash
cd rpm-build
# Build the certificates package
./build-cert-rpm.sh
# Build the server package
./build-rpm.sh
```

(if the cert package is already built, you can skip this step)


## Tests

```bash
$ go test
PASS
ok      github.com/palette-software/insight-server      0.064s
```


## gofmt pre-commit hook:

Go has a formatting tool that formats all code to the official go coding standard, called ```gofmt```. From the [go documentation](https://github.com/golang/go/wiki/CodeReviewComments#gofmt):

> Run gofmt on your code to automatically fix the majority of mechanical style issues. Almost all Go code in the wild uses gofmt. The rest of this document addresses non-mechanical style points.
>
> An alternative is to use goimports, a superset of gofmt which additionally adds (and removes) import lines as necessary.

To use this tool before each commit, create the following ```.git/hooks/pre-commit``` file:

```bash
#!/bin/sh
# Copyright 2012 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# git gofmt pre-commit hook
#
# To use, store as .git/hooks/pre-commit inside your repository and make sure
# it has execute permissions.
#
# This script does not handle file names that contain spaces.

gofiles=$(git diff --cached --name-only --diff-filter=ACM | grep '.go$')
[ -z "$gofiles" ] && exit 0

unformatted=$(gofmt -l $gofiles)
[ -z "$unformatted" ] && exit 0

# Some files are not gofmt'd. Print message and fail.

echo >&2 "Go files must be formatted with gofmt. Please run:"
for fn in $unformatted; do
	echo >&2 "  gofmt -w $PWD/$fn"
done

exit 1
```

TODO: add this check to travis


