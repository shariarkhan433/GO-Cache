package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Cache struct {
	mu   sync.RWMutex      //the lock ?
	data map[string]string //the hash map
}

var store = Cache{
	data: make(map[string]string),
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
	reader := bufio.NewReader(conn)

	for {
		// USE THE NEW PARSER HERE
		parts, err := parseRESP(reader)
		if err != nil {
			// If error is EOF, client disconnected
			return
		}

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		fmt.Printf("Processing command: %v\n", parts) // Debug print

		switch command {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))

		case "SET":
			if len(parts) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
				continue
			}
			key, value := parts[1], parts[2]

			store.mu.Lock()
			store.data[key] = value
			store.mu.Unlock()

			conn.Write([]byte("+OK\r\n"))

		case "GET":
			if len(parts) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := parts[1]

			store.mu.RLock()
			val, ok := store.data[key]
			store.mu.RUnlock()

			if !ok {
				conn.Write([]byte("$-1\r\n"))
			} else {
				resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
				conn.Write([]byte(resp))
			}

		// Handle the initial "COMMAND" check from redis-cli
		case "COMMAND":
			conn.Write([]byte("+OK\r\n"))

		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

// Helper to read a line until \r\n and return it without the \r\n
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	// Strip \r\n (Windows/Internet standard) or just \n
	return strings.TrimRight(line, "\r\n"), nil
}

// The Core Parser
func parseRESP(reader *bufio.Reader) ([]string, error) {
	// 1. Read the first byte to check if it's an Array (*)
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if prefix != '*' {
		// If it's not an array (e.g. inline command), treat as simple line
		// This is a fallback for simple "PING" or "SET key val" via nc
		reader.UnreadByte() // Put the byte back
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		return strings.Fields(line), nil
	}

	// 2. Read the number of elements (Array Length)
	lenStr, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	// Convert string "3" to int 3
	var numElements int
	fmt.Sscanf(lenStr, "%d", &numElements)

	args := make([]string, 0, numElements)

	// 3. Loop to parse each element
	for i := 0; i < numElements; i++ {
		// Expect '$' for Bulk String
		prefix, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		if prefix == '$' {
			// Read length of the string
			lengthStr, _ := readLine(reader)
			var length int
			fmt.Sscanf(lengthStr, "%d", &length)

			// Read the actual data (plus the \r\n at the end)
			// We use length+2 to consume the trailing \r\n
			data := make([]byte, length+2)
			reader.Read(data)

			// Store just the string (remove \r\n)
			args = append(args, string(data[:length]))
		}
	}

	return args, nil
}
