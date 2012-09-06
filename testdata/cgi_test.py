import sys

def echo(environ, start_response):
    start_response('200 OK', [('Content-Type', 'text/plain')])
    def resp():
        for l in environ['wsgi.input'].readlines():
            yield l

    return resp()

if __name__ == '__main__':
    from flup.server.fcgi import WSGIServer
    WSGIServer(echo, bindAddress = ('127.0.0.1', 9000)).run()


