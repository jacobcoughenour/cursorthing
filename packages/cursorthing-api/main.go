package main

func main() {
	server := NewServer()
	err := server.ListenAndServe(8080)

	if err != nil {
		panic(err)
	}
}
