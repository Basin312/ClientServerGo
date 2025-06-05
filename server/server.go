package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// sturcture identity user
type User struct {
	name    string
	koneksi net.Conn
}

// untuk simpan user yang login (Resource: need mutex if multiple user want to change or append)
var listUser = []User{}

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

	for {
		// menerima koneksi client
		conm, err := listenConection.Accept()

		// jika tidak bisa menerima maka error
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection!")
			os.Exit(1)
		} else {
			fmt.Println("New connection accepted!")
		}

		go communicationWithServer(conm)
	}

}

func communicationWithServer(koneksiUser net.Conn) {

	// menerima pesan dari client
	reader := bufio.NewReader(koneksiUser)

	fmt.Println("Waiting for user insert name")
	// check jika berhasil membaca maka
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read name: %v\n", err)
		return
	}

	name = strings.TrimSpace(name)
	fmt.Printf("User '%s' has joined.\n", name)

	// Register user
	user := User{
		name:    name,
		koneksi: koneksiUser,
	}

	listUser = append(listUser, user)

	// Inform user
	fmt.Fprintf(koneksiUser, "Welcome, %s! Type a message:\n", name)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("User '%s' disconnected.\n", name)
			return
		}

		message = strings.TrimSpace(message)
		fmt.Printf("[%s] %s\n", user.name, message)

		// Echo back to the client
		fmt.Fprintf(koneksiUser, "Echo [%s]: %s\n", user.name, message)
	}

}
