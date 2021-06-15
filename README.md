# ably-protocols-test

This is meant as a demo for a basic protocol proof of concept over TCP.

It's a bit rough and ready at the moment. I haven't tested it that thoroughly due to time constraints.

If you have trouble running it, or find errors, please let me know.

## Usage

- `make install` to install dependencies

- `make server` to run the server (make sure nothing's listening on port 8080 or 8081)

- `make stateless` to run the stateless request
- `make stateful` to run the stateful request

## Architecture

The server is written in Go. The client is written in NodeJS using Typescript.
