install:
	@yarn install

server:
	go run ./src

stateless:
	./node_modules/.bin/ts-node src/client --stateless

stateful:
	./node_modules/.bin/ts-node src/client --stateful
