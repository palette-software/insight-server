[![Build Status](https://travis-ci.com/palette-software/insight-server.svg?token=qWG5FJDvsjLrsJpXgxSJ&branch=master)](https://travis-ci.com/palette-software/insight-server)

# Palette Insight Webservice

## What is Palette Insight Server

This component is responsible for receiving data from the agents on the Tableau Servers and storing that data in a format that is compatible with the database importing component.

# How do I set up Palette Insight Server

## Prerequisites

  * Operating system: CentOS/RHEL 6.5+
  * The server is using [Supervisord](http://supervisord.org/installing.html#installing-to-a-system-with-internet-access) for daemonizitation.

## Installation

### From rpm.palette-software.com

Make sure there is a repository definition file pointing to Palette RPM Repository:
```
/etc/yum.repos.d/palette.repo
```

Contents:

```
          [palette-rpm]
          name=Palette RPM
          baseurl=https://rpm.palette-software.com/centos/dev
          enabled=1
          gpgcheck=0
```

Install palette-insight-server
```yum install palette-insight-server```

## Contribution

### Building locally

```
go get ./...
go build -v
```

### Testing

```go test ./... -v``` 

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
./insight-server.exe --help
Usage of C:\Users\Miles\go\src\github.com\palette-software\insight-server\server\insigh-server.exe:
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

A sample configuration file can be found in ```sample.config```

```
# PATHS
# =====

# The root directory for the uploads to go into.
upload_path=/data/insight-server/uploads

# The path where the maxid files are stored
maxid_path=/data/insight-server/maxids

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

```

This configuration file gets installed as default when using the RPM installer.

## IpTables

To allow the service to listen to port 443 without sudo privileges an IpTables forwarding needs to be set up.

```bash
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 443 -j REDIRECT --to-port 9443
```

