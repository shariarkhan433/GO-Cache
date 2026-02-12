package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

type AOF struct {
	file *os.File
	mu   sync.Mutex // Protects the file write
}

func NewAOF(path string) (*AOF, error) {
	// Open file: Create if missing, Append mode, Read/Write
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &AOF{file: f}, nil
}

func (aof *AOF) Close() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	return aof.file.Close()
}

// Write writes the command to the AOF file
// Example: Write("SET", "key", "value") -> "*3\r\n$3\r\nSET\r\n..."
func (aof *AOF) Write(args ...string) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	// Reconstruct the RESP array
	// 1. Array Header (*N)
	// We use a buffer or just multiple writes.
	// Since you are learning, let's keep it simple with Fprintf.
	// Ideally, you'd use a buffer to minimize syscalls.

	// Write "*3\r\n"
	_, err := aof.file.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	if err != nil {
		return err
	}

	for _, arg := range args {
		// Write "$3\r\nSET\r\n"
		_, err := aof.file.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
		if err != nil {
			return err
		}
	}

	return nil
}

func (aof *AOF) Read(fn func(value []string)) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	// Go back to the beginning of the file
	aof.file.Seek(0, 0)

	reader := bufio.NewReader(aof.file)

	for {
		// Use your existing parseRESP function!
		// Note: You might need to make parseRESP public or copy it here.
		// For now, assume it's available.
		p, err := parseRESP(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		fn(p) // Callback: Execute the command
	}

	return nil
}
