#!/usr/bin/env python3

import sys

from http import HTTPStatus
from http.server import HTTPServer, BaseHTTPRequestHandler


ADDR = ""
PORT = 8080


class DoorBotRequest(BaseHTTPRequestHandler):
    def do_POST(self):
        self.log_message("request received")
        file_length = int(self.headers["Content-Length"])
        with open(sys.argv[1], "ab") as f:
            f.write(self.rfile.read(file_length))
            f.write("\n".encode())
        self.send_response(HTTPStatus.OK)
        self.end_headers()


def run(server_class=HTTPServer, handler_class=DoorBotRequest):
    server_address = (ADDR, PORT)
    httpd = server_class(server_address, handler_class)
    httpd.serve_forever()


def main() -> int:
    run()
    return 0


if __name__ == "__main__":
    sys.exit(main())
