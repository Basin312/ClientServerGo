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

	// Send username
	for {
		// Read initial prompt from server
		for i := 0; i < 6; i++ {
			line, err := reader.ReadString('\n')
			if err != nil{
				fmt.Println("Gagal membaca pesan selamat datang")
				return
			}
			fmt.Print(line)
		}

		username, _ := stdin.ReadString('\n')
		conn.Write([]byte(username))

		// Expect either "ok" or "taken"
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Failed to read username response")
			continue
		}

		response = strings.TrimSpace(response)
		if response == "ok" {
			break
		}
		if response == "taken" {
			message, _ := reader.ReadString('\n')
			fmt.Println(message)
		}
	}

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
