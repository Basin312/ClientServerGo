package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// variabel global untuk client
var (
	wg      sync.WaitGroup
	name    string
	err     error
	koneksi net.Conn
)

func main() {
	// Koneksi ke server di localhost:9090
	koneksi, err = net.Dial("tcp", ":9090")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot connect to server!")
		os.Exit(1)
	}
	fmt.Println("Connected to server!")

	// Reader untuk koneksi dan stdin
	connReader := bufio.NewReader(koneksi)
	localReader := bufio.NewReader(os.Stdin)

	wg.Add(1)
	// Terima pesan dari server (misal: "masukan nama")
	go receivedMessage(connReader)
	// Jalankan goroutine untuk mengirim pesan
	go sentMessage(localReader, koneksi)

	wg.Wait()
}

func insertNamePhase(conn net.Conn, connReader bufio.Reader, local *bufio.Reader) {
	for {
		// Masukan nama
		fmt.Print("Enter your username: ")
		name, err = local.ReadString('\n')

		// Mencegah error dan nama kosong
		if err != nil {
			fmt.Fprintln(os.Stderr, "=== Cannot read the name! ===")
			continue
		} else if strings.TrimSpace(name) == "" {
			fmt.Println("=== Name cannot be empty ===")
			continue
		}

		conn.Write([]byte(name))
		name = strings.Trim(name, "\r\n")
	}
}
func sentMessage(localReader *bufio.Reader, conn net.Conn) {
	for {

		message, err := localReader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "=== Sorry, we encounter an error ===")
			break
		}

		if strings.TrimSpace(message) == "" {
			continue
		}

		message = message + "\n"
		conn.Write([]byte(message))
	}
}

func receivedMessage(connReader *bufio.Reader) {
	for {
		message, err := connReader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "=== Connection closed ===")
			os.Exit(1)
		}
		fmt.Print(message) // langsung print tanpa newline
	}
}
