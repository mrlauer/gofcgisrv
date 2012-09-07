gofcgisrv
=========

Go package for the webserver end of the
[CGI](http://tools.ietf.org/html/rfc3875), 
[FastCGI](http://www.fastcgi.com/drupal/node/6?q=node/22), 
and [SCGI](http://python.ca/scgi/protocol.txt) protocols.

The terms "server" and "client" are confusing. "Server" here generally means "webserver," as referred to in
(for example) the FastCGI spec. In terms of who is dialing whom, the webserver is the FastCGI or SCGI client.
Sorry.

See godoc for usage.

No one really seems to support FastCGI properly and completely.

Bugs and todos
--------------

There is nothing here to launch processes. Only TCP connections are supported, not STDIN.

Not all CGI headers are correctly set.
