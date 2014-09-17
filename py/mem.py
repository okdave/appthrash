import webapp2
import time
import threading

from google.appengine.api import memcache


class MainPage(webapp2.RequestHandler):
    def get(self):
        self.response.headers['Content-Type'] = 'text/plain'
        self.response.write("""Paths
\t/mem?count=N - fire N concurrent memcache.Get requests
""")


class MemPage(webapp2.RequestHandler):
    def get(self):
        count = int(self.request.get('count'))
        if count < 1:
            self.response.write('Count must be at least 1\n')
            return

        rpcs = []
        c = memcache.Client()

        for x in xrange(count):
            rpc = memcache.create_rpc(deadline=30)
            c.get_multi_async(["roger"], rpc=rpc)
            rpcs.append(rpc)

        for rpc in rpcs:
            rpc.wait()

        self.response.write('OK\n')


application = webapp2.WSGIApplication([
    ('/', MainPage),
    ('/mem', MemPage),
], debug=True)
