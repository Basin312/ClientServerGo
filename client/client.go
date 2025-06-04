package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// bikin koneksi type network dan alamat port
	conn, err := net.Dial("tcp", ":9090")

	//check kalau ada error atau tidak, 
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!")
	} else {
		fmt.Println("Connected to server!")
	}

    // untuk menerima pesan dari server
	connReader := bufio.NewReader(conn)
    // input reader untuk keyboard
	localReader := bufio.NewReader(os.Stdin)
    
	fmt.Print("Type your message> ")
	
    // membaca localreader sampai user klik enter
    message, err := localReader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read the message!")
	} else {
		fmt.Println("The message has been read!")
	}

	conn.Write([]byte(message)) // fmt.Fprint(conn, message)
	fmt.Println("The message has been sent!")
	fmt.Println("Waiting for reply...")
	echo, err := connReader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read the echo!")
		os.Exit(1)
	} else {
		fmt.Println("The echo has been received!")
	}

	fmt.Println(echo)
}
