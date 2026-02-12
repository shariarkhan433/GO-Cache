package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Cache struct {
	mu      sync.RWMutex
	data    map[string]string
	expires map[string]int64
}

var store = Cache{
	data:    make(map[string]string),
	expires: make(map[string]int64),
}

// FIX 1: Make 'aof' global so handleConnection can see it
var aof *AOF

func main() {
	var err error
	// Initialize the global 'aof' variable
	aof, err = NewAOF("database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aof.Close()

	// FIX 3: Update Replay Logic to handle deletions
	aof.Read(func(value []string) {
		command := strings.ToUpper(value[0])

		if command == "SET" {
			key, val := value[1], value[2]
			store.mu.Lock()
			store.data[key] = val
			store.mu.Unlock()
		} else if command == "DEL" {
			key := value[1]
			store.mu.Lock()
			delete(store.data, key)
			store.mu.Unlock()
		}
	})

	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	fmt.Println("Listener on port :6379")

	go startJanitor()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		parts, err := parseRESP(reader)
		if err != nil {
			return
		}

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		// fmt.Printf("Processing command: %v\n", parts)

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

			// Write to AOF
			aof.Write("SET", key, value)
			conn.Write([]byte("+OK\r\n"))

		case "GET":
			if len(parts) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := parts[1]

			store.mu.Lock()
			exp, hasExpiry := store.expires[key]
			if hasExpiry && time.Now().Unix() > exp {
				delete(store.data, key)
				delete(store.expires, key)
				store.mu.Unlock()
				conn.Write([]byte("$-1\r\n"))
				continue
			}

			val, ok := store.data[key]
			store.mu.Unlock()

			if !ok {
				conn.Write([]byte("$-1\r\n"))
			} else {
				resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
				conn.Write([]byte(resp))
			}

		case "EXPIRE":
			if len(parts) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'expire' command\r\n"))
				continue
			}
			key := parts[1]
			secondsStr := parts[2]

			var seconds int64
			fmt.Sscanf(secondsStr, "%d", &seconds)

			store.mu.Lock()
			store.expires[key] = time.Now().Unix() + seconds
			store.mu.Unlock()

			conn.Write([]byte(":1\r\n"))

		case "DEL":
			// FIX 2: Implement the DEL logic completely
			if len(parts) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'del' command\r\n"))
				continue
			}
			key := parts[1]

			store.mu.Lock()
			delete(store.data, key)
			store.mu.Unlock()

			// Write to AOF
			aof.Write("DEL", key)
			conn.Write([]byte(":1\r\n")) // Redis returns integer 1 for success

		case "COMMAND":
			conn.Write([]byte("+OK\r\n"))

		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

// ... [Keep your helper functions parseRESP and readLine here] ...
// ... [Keep startJanitor here] ...
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

func startJanitor() {
	for {
		// Sleep for 1 second
		time.Sleep(1 * time.Second)

		// Wake up and scan for dead keys
		now := time.Now().Unix()

		store.mu.Lock()
		for key, exp := range store.expires {
			if now > exp {
				delete(store.data, key)
				delete(store.expires, key)
				fmt.Printf("Janitor: Cleaned up %s\n", key) // Log to server console
			}
		}
		store.mu.Unlock()
	}
}
