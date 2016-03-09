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
| bool   | -tls                                       | TLS=true                                  | tls=true                                  |
| string | -cert certs/cert.pem                       | CERT=certs/cert.pem                       | cert=certs/cert.pem                       |
| string | -key certs/key.pem                         | KEY=certs/key.pem                         | cert=certs/key.pem                        |

To get a list of command line options, use the ```--help``` switch. On my machine (windows) this results in:

```
Usage of C:\Users\Miles\go\src\github.com\palette-software\insight-server\server\server.exe:
  -bind_address="": The address to bind to. Leave empty for default .
  -cert="cert.pem": The TLS certificate file to use when tls is set.
  -config="": Configuration file to use.
  -key="key.pem": The TLS certificate key file to use when tls is set.
  -licenses_path="C:\\Users\\Miles\\go\\src\\github.com\\palette-software\\insight-server\\server\\licenses": The directory the licenses are loaded from on start.
  -maxid_path="C:\\Users\\Miles\\AppData\\Local\\Temp\\uploads\\maxid": The root directory for the maxid files to go into.
  -port=9000: The port the server is binding itself to
  -tls=false: Use TLS for serving through HTTPS.
  -upload_path="C:\\Users\\Miles\\AppData\\Local\\Temp\\uploads": The root directory for the uploads to go into.
```

## Sample configuration file

A sample configuration file from one of the test machines:

```
upload_path=/mnt/dbdata/insight-server/uploads
maxid_path=/mnt/dbdata/insight-server/maxids
licenses_path=/mnt/dbdata/insight-server/licenses
port=9443
tls=true
#cert=/mnt/dbdata/insight-server/ssl-certs/server.crt
cert=/mnt/dbdata/insight-server/ssl-certs/star_palette-software_net.crt
key=/mnt/dbdata/insight-server/ssl-certs/server.key
```

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

## API Documentation

A basic documentation using OpenAPI is available in the docs folder, or
a HTML-ized version is available in the docs/generated folder.

## Tests

```bash
$ go test
PASS
ok      github.com/palette-software/insight-server      0.064s
```
