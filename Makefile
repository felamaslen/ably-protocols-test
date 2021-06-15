server:
	go run ./src/server.go

client:
	./node_modules/.bin/ts-node src/client.ts
