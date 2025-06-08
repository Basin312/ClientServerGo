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

	reader := bufio.NewReader(conn)    // Membaca data dari server
	stdin := bufio.NewReader(os.Stdin) // Membaca input pengguna

	// Mengirim username sampai server mengonfirmasi
	for {
		// Membaca initial prompt dari server
		for i := 0; i < 6; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Gagal membaca pesan selamat datang")
				return
			}
			fmt.Print(line)
		}

		// Membaca nama pengguna dari terminal dan mengirim ke server
		username, _ := stdin.ReadString('\n')
		conn.Write([]byte(username))

		// Mengharapkan respons "ok" atau "taken"
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

	// Menampilkan pesan sambutan dari server
	welcome, _ := reader.ReadString('\n')
	fmt.Print(welcome)

	// Start goroutine untuk membaca pesan dari server
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\033[33m\nDisconnected from server.\033[0m")
				os.Exit(0)
			}
			fmt.Print(message)

		}
	}()

	// Main input loop, membaca input pengguna dan mengirim ke server
	for {
		text, _ := stdin.ReadString('\n')
		text = strings.TrimSpace(text)
		if text != "" {
			conn.Write([]byte(text + "\n"))
		}
	}
}
