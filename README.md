# Notes from T

## Adding a user from a license

To add a user to the authorized users on the server, navigate your
browser to the following url:

```
http://<ENDPOINT>/users/new
```

This form allows you to create a user from a license. On success, the
server returns a JSON object of the newly created user.

## Running the tests

Run the tests by:

```bash
revel test github.com/palette-software/insight-server test
```


And run a test server by

```bash
revel run github.com/palette-software/insight-server dev
```

The test runner is available then at:

```
http://localhost:9000/@tests
```

Or run in prod with HTTPS enabled (certs are in the root directory)
with:

```bash
revel run github.com/palette-software/insight-server prod
```


More about the HTTPS cert process can be found in the

```
server.key.info
```

file.



## API Documentation


A basic documentation using OpenAPI is available in the docs folder, or
a HTML-ized version is available in the docs/generated folder.

