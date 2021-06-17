install:
	@yarn install

server:
	go run ./src 8080

stateless:
	./node_modules/.bin/ts-node src/client 8080 --stateless

stateful:
	./node_modules/.bin/ts-node src/client 8080 --stateful
