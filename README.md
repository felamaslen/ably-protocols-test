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

## Protocol description

### Stateless

The client initiates a TCP connection with the server on port 8080.
The client should write the following data:

`xxxyyyyyzzzzz`

where:

- `xxx` is the zero-padded value of `a`, which should be `000` in the case of the initial request
- `yyyyy` is the zero-padded value of `n` which specifies the number of *remaining* numbers required
- `zzzzz` is the zero-padded value of `m` which specifies how many numbers have already been received, in the case of *subsequent requests* (after failures)

The client is responsible for keeping track of `a`, `n`, and `m`. It is also responsible for retrying failed connections using exponential backoff or a strategy of its choice.

In the case of the written client, failed requests trigger a recursive, exponentially backed-off call to the request initiator, using the updated values of `a`, `n`, and `m` so the server knows where to resume from.

The client knows the stream is complete when it receives the following data: `EOF\n`.

### Stateful

The client initiates a TCP connection with the server on port 8081.
The client should generate a UUID, and write the following data:

`yyyyyUUUUUUUU-UUUU-UUUU-UUUU-UUUUUUUUUUUU`

where:

- `yyyyy` is the zero-padded value of `n` which specifies the number of numbers required
- `UUU...` is the generated UUID

The same exponential backoff retry strategy is employed in the stateful as the stateless case, except the values of `n` and the uuid (i.e. the reconnection parameters) do not change each time.

The same EOF line for the end of response is used in the stateful as the stateless case.

The client can validate the result in the stateless case by calculating the checksum and comparing it to the response given, where the response checksum is given in a message like `response=checksum\n`.

#### PRNG

The golang `math/rand` library is used to generate a uint32 array, using `rand.Uint32()`. The PRNG is seeded during the server's startup, using the current UNIX nanosecond timestamp.

#### Session state

The session state for each client includes:

- `uuid` - necessary to identify the client from the incoming request
- `seq` - sequence of numbers generated for this client; we don't want this to change between requests
- `progress` - how far along the sequence the client has already received; necessary so we don't replay the entire sequence when requests fail
- `selfDestructTimer` - this is an implementation detail; if I was using redis I would use its native expiry feature so that the session data is destroyed after a timeout. In golang I am using a timer for this.

There is also the value of `n` which was passed from the client; I don't think this is necessary on second thoughts, as it can be calculated from the length of `seq` anyway.

#### Checksum

The checksum used to verify that the data received were correct, is an `md5` hash of the JSON-encoded array of numbers. So for example: `md5([16723,998128,85773])`.
