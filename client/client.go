package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", ":9090")
	if err != nil {
		fmt.Println("Cannot connect to server:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	stdin := bufio.NewReader(os.Stdin)

	// Read initial prompt from server
	prompt, _ := reader.ReadString(':')
	fmt.Print(prompt)

	// Send username
	username, _ := stdin.ReadString('\n')
	conn.Write([]byte(username))

	// Display welcome message
	welcome, _ := reader.ReadString('\n')
	fmt.Print(welcome)

	// Start goroutine to read from server
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected from server.")
				os.Exit(0)
			}
			fmt.Print(message)
		}
	}()

	// Main input loop
	for {
		text, _ := stdin.ReadString('\n')
		text = strings.TrimSpace(text)
		if text != "" {
			conn.Write([]byte(text + "\n"))
		}
	}
}
