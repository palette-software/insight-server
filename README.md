# Palette Insight Webservice

## Starting the webservice

```bash
cd server
go build && ./server
```

or on Windows:

```
cd server
go build
server.exe
```

This starts the webservice on:

- if the ```PORT``` environment variable is set, then on the port specified by that
- if it isnt set, then on port 9000


## User authentication

On startup, the server tries to load all licenses from:

- if the ```INSIGHT_LICENSES_PATH``` environment variable is set, then from the directory it points to
- if it isnt set, the license files are loaded from the 'licenses' subdirectory inside the server executables directory

The usernames are the ```licenseId``` field of the license, the authentication token is the ```token``` field of the license.

## Setting the upload path

The ```INSIGHT_UPLOAD_HOME``` environment variable describes the root directory where the uploads are kept. If its not
set, the ```$TEMP/uploads`` is used. 

For example:

```bash
INSIGHT_UPLOAD_HOME=/opt/insight-server/uploads PORT=8080 INSIGHT_LICENSES_PATH=/opt/insight-server/licenses ./server
```

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


The maxids are stored in:

* The directory pointed to by the ```INSIGHT_MAXID_PATH``` or the ```_maxid``` subfolder of the upload base directory.


## Checking if the service is running

```
$ curl http://localhost:9000/
PONG
```

## Uploading a file

See the openAPI documentation inside the docs/generated folder

## API Documentation

A basic documentation using OpenAPI is available in the docs folder, or
a HTML-ized version is available in the docs/generated folder.

## Configuration

The webservice uses the 'flag' library to handle configuration via both configuration files and environment variables.

For more information on the flag library see [github.com/namsral/flag](https://github.com/namsral/flag).

The environment variables and their corresponding configuration file values and command line flags are:


| Type   | Flag                                       | Environment                               | File                                      |
|--------|--------------------------------------------|-------------------------------------------|-------------------------------------------|
| string | -upload_path=/opt/insight-agent/uploads    | UPLOAD_PATH=/opt/insight-agent/uploads    | upload_path=/opt/insight-agent/uploads    |
| string | -maxid_path=/opt/insight-agent/maxids      | MAXID_PATH=/opt/insight-agent/maxids      | maxid_path=/opt/insight-agent/maxids      |
| string | -licenses_path=/opt/insight-agent/licenses | LICENSES_PATH=/opt/insight-agent/licenses | licenses_path=/opt/insight-agent/licenses |
| string | -config dev.config                         | CONFIG=dev.config                         | config=dev.config                         |
| int    | -bind_port 8080                            | BIND_PORT=8080                            | bind_port=8080                            |
| string | -bind_address 127.0.0.1                    | BIND_ADDRESS=127.0.0.1                    | bind_address=127.0.0.1                    |

## Tests

Due to the quick pace of the development, the existing tests have been scrapped for the most part.

Running them:

```bash
$ go test
PASS
ok      github.com/palette-software/insight-server      0.064s
```
