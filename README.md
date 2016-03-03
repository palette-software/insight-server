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


## Tests

Due to the quick pace of the development, the existing tests have been scrapped for the most part.

Running them:

```bash
$ go test
PASS
ok      github.com/palette-software/insight-server      0.064s
```
