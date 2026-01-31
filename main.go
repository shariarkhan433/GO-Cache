package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Cache struct{
	mu sync.RWMutex //the lock ?
	data map[string]string //the hash map
}

store := Cache{
	data : make(map[string]string),
}

func main() {
	//Listen on a TCP port (socket programming)
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	fmt.Println("Listener on port :6379")

	for {
		//Accepting new connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		//handling concurrency
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// create a buffer to read the stream
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		//stripe white space from the command
		message = strings.TrimSpace(message)
		fmt.Printf("Received: %s\n", message)

		//Echo it back for now
		conn.Write([]byte("you said: " + message + "\n"))
	}
}
