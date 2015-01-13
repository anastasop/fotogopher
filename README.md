
# Fotogopher

A small toy project that provides an API for taking screenshots of web sites, similar to [urlbox](http://urlbox.io)

Written in [go](http://www.golang.org), it demonstrates the simplicity and expressiveness of the language for writing concurrent applications.

## Installation

1. Install [go](http://golang.org/)
1. Install timeout from GNU coreutils. Probably it is already installed in your system
2. Install [phantom.js](http://phantomjs.org/) and put the binary in your PATH
3. `go run fotogopher.go --help` for a view of the runtime options
4. `go run fotogopher.go` to start the server. It must run from its directory so that it can find `capture.js`
5. go to http://localhost:8080 to read the documentation of the API

Tested with go1.4 and phantomjs 1.9.8 on ubuntu LTS 12.04
