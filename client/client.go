//Client

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	//Membuat koneksi ke server
	conn, err := net.Dial("tcp", ":9090")

	if err != nil {
		fmt.Println("Cannot connect to server:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)    //Membaca data dari server
	stdin := bufio.NewReader(os.Stdin) //Membaca input client

	//Loop hingga username client diterima (unik)
	for {
		//Loop 6x, sambutan welcome ada 6 baris
		for range 6 {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Gagal membaca pesan selamat datang")
				return
			}
			fmt.Print(line)
		}

		//Input nama client dan kirim ke server
		username, _ := stdin.ReadString('\n')
		conn.Write([]byte(username))

		//Respon validasi dari server
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Failed to read username response")
			continue
		}

		response = strings.TrimSpace(response)
		if response == "ok" {
			break
		}

		if response == "taken" { //Nama sudah digunakan
			message, _ := reader.ReadString('\n')
			fmt.Println(message)
		}
	}

	//Sambutan dari server
	welcome, _ := reader.ReadString('\n')
	fmt.Print(welcome)

	//Goroutine untuk membaca pesan dari server
	go func() {
		//Loop terus untuk mendengarkan dan menampilkan pesan dari server
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\033[33m\nDisconnected from server.\033[0m")
				os.Exit(0)
			}
			fmt.Print(message)
		}
	}()

	//Loop terus, membaca input client dan kirim ke server
	for {
		text, _ := stdin.ReadString('\n')
		text = strings.TrimSpace(text)
		if text != "" {
			conn.Write([]byte(text + "\n"))
		}
	}
}