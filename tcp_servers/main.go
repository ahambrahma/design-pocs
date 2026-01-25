package main

func main() {
	//Listen()
	// ListenMultiThreaded()
	if err := ListeningForConnections(); err != nil {
		panic(err)
	}
}
