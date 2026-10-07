#!/usr/bin/env python3
"""Loopback TLS test endpoint forwarding to the separately running local model."""
import argparse
import http.client
import http.server
import ssl
import time

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--certificate',required=True)
parser.add_argument('--key',required=True)
parser.add_argument('--port',type=int,default=18443)
parser.add_argument('--upstream-port',type=int,default=18081)
parser.add_argument('--trace-stream', action='store_true', help='log completion timing and byte counts without response contents')
args=parser.parse_args()

class Proxy(http.server.BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def forward(self):
        started, transferred = time.monotonic(), 0
        body=self.rfile.read(int(self.headers.get('Content-Length','0')))
        upstream=http.client.HTTPConnection('127.0.0.1',args.upstream_port,timeout=360)
        try:
            headers={key:value for key,value in self.headers.items()
                     if key.lower() not in ('host','connection','transfer-encoding','content-length')}
            upstream.request(self.command,self.path,body=body,headers=headers)
            response=upstream.getresponse()
            self.send_response(response.status)
            for key,value in response.getheaders():
                if key.lower() not in ('connection','transfer-encoding','content-length'):
                    self.send_header(key,value)
            self.send_header('Transfer-Encoding','chunked')
            self.end_headers()
            while True:
                chunk=response.read1(4096)
                if not chunk: break
                transferred += len(chunk)
                self.wfile.write(('%x\r\n'%len(chunk)).encode()+chunk+b'\r\n')
                self.wfile.flush()
            self.wfile.write(b'0\r\n\r\n');self.wfile.flush()
            if args.trace_stream:
                print('TLS_PROXY_STREAM_END id=%d path=%s status=%d bytes=%d seconds=%.3f' %
                      (id(self), self.path, response.status, transferred, time.monotonic()-started), flush=True)
        finally:
            upstream.close()
    do_GET=forward
    do_POST=forward

context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain(args.certificate,args.key)
server=http.server.ThreadingHTTPServer(('127.0.0.1',args.port),Proxy)
server.socket=context.wrap_socket(server.socket,server_side=True)
print('Local model TLS proxy ready on 127.0.0.1:%d'%args.port,flush=True)
server.serve_forever()
