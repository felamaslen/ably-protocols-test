package main

func main() {
	go listenStateless()
	go listenStateful()

	select {}
}
