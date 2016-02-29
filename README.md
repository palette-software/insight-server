# Notes from T

## Adding a user from a license

To add a user to the authorized users on the server, navigate your
browser to the following url:

```
http://<ENDPOINT>/users/new
```

This form allows you to create a user from a license. On success, the
server returns a JSON object of the newly created user.

## Adding a test user for testing with the agent:

There is an endpoint to create a tenant with the credentials:

```
username: test
password: test
```

that can be created by:

```bash
curl http://localhost:9000/users/create-test
```

Currently each time you call this endpoint, a new tenant gets added (until
username uniqueness validation is added).

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

# ORIGINAL README FOLLOWS

--


# Welcome to Revel

## Getting Started

A high-productivity web framework for the [Go language](http://www.golang.org/).

### Start the web server:

    revel run myapp

   Run with <tt>--help</tt> for options.

### Go to http://localhost:9000/ and you'll see:

"It works"

### Description of Contents

The default directory structure of a generated Revel application:

    myapp               App root
      app               App sources
        controllers     App controllers
          init.go       Interceptor registration
        models          App domain models
        routes          Reverse routes (generated code)
        views           Templates
      tests             Test suites
      conf              Configuration files
        app.conf        Main configuration file
        routes          Routes definition
      messages          Message files
      public            Public assets
        css             CSS files
        js              Javascript files
        images          Image files

app

    The app directory contains the source code and templates for your application.

conf

    The conf directory contains the application’s configuration files. There are two main configuration files:

    * app.conf, the main configuration file for the application, which contains standard configuration parameters
    * routes, the routes definition file.


messages

    The messages directory contains all localized message files.

public

    Resources stored in the public directory are static assets that are served directly by the Web server. Typically it is split into three standard sub-directories for images, CSS stylesheets and JavaScript files.

    The names of these directories may be anything; the developer need only update the routes.

test

    Tests are kept in the tests directory. Revel provides a testing framework that makes it easy to write and run functional tests against your application.

### Follow the guidelines to start developing your application:

* The README file created within your application.
* The [Getting Started with Revel](http://revel.github.io/tutorial/index.html).
* The [Revel guides](http://revel.github.io/manual/index.html).
* The [Revel sample apps](http://revel.github.io/samples/index.html).
* The [API documentation](http://revel.github.io/docs/godoc/index.html).

## Contributing
We encourage you to contribute to Revel! Please check out the [Contributing to Revel
guide](https://github.com/revel/revel/blob/master/CONTRIBUTING.md) for guidelines about how
to proceed. [Join us](https://groups.google.com/forum/#!forum/revel-framework)!
