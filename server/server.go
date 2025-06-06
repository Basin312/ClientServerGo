package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Client struct {
	conn net.Conn
	name string
}

var (
	clients   = make(map[net.Conn]Client) // koneksi aktif
	broadcast = make(chan string)         // channel untuk siaran pesan
	lock      sync.Mutex                  // untuk sinkronisasi akses map
)

func main() {
	// membuat koneksi untuk mendengar protocol tcp alamat 9090
	listenConection, err := net.Listen("tcp", ":9090")

	// check jka error membuat connection akan keluar pesan
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen!")
		os.Exit(1)
	} else {
		fmt.Println("Listening...")
	}

	go broadcaster()

	// menerima koneksi client
	for {
		conn, err := listenConection.Accept()

		// jika tidak bisa menerima maka error
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection!")
			os.Exit(1)
		} else {
			fmt.Println("New connection accepted!")
		}
		go handleClient(conn)
	}

}
func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	lock.Lock()
	clients[conn] = Client{conn: conn, name: name}
	lock.Unlock()

	broadcast <- fmt.Sprintf(">> %s has joined the chat\n", name)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		message = strings.TrimSpace(message)
		if message != "" {
			broadcast <- fmt.Sprintf("[%s]: %s\n", name, message)
		}
	}
	lock.Lock()
	delete(clients, conn)
	lock.Unlock()
	broadcast <- fmt.Sprintf(">> %s has left the chat\n", name)
}

func broadcaster() {
	for {
		msg := <-broadcast
		lock.Lock()
		for conn := range clients {

			go func(c net.Conn) {
				_, err := c.Write([]byte(msg))
				if err != nil {
					c.Close()
					delete(clients, c)
				}
			}(conn)
		}
		lock.Unlock()
	}
}
