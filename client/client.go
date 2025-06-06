package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func listenMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from server.")
			os.Exit(0)
		}
		fmt.Print(msg)
	}
}

func main() {
	// bikin koneksi type network dan alamat port
	conn, err := net.Dial("tcp", ":9090")

	//check kalau ada error atau tidak,
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!")
	} else {
		fmt.Println("Connected to server!")
	}

	// mendengarkan pesan
	go listenMessages(conn)

	//input username dari client
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	conn.Write([]byte(name + "\n"))

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "/exit" {
			break
		}
		conn.Write([]byte(text + "\n"))

	}
}
