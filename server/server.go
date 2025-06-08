package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type Client struct {
	name     string
	conn     net.Conn
	incoming chan string
	room     string
}

type Message struct {
	from    string
	room    string
	content string
}

var (
	clients     = make(map[net.Conn]*Client) // List client yang ada di server
	rooms       = make(map[string][]*Client) // List room dan client yang ada di room tersebut
	broadcast   = make(chan Message)         // Channel untuk pesan broadcast
	lock        = sync.Mutex{}
	logger      *log.Logger
	helpMessage = "\033[33m" +
		"\n+---------------------------------------------+\n" +
		"|  🔧 Commands you can use:                   |\n" +
		"|   • /join <room>   → Join or create room    |\n" +
		"|   • /leave         → Leave current room     |\n" +
		"|   • /rooms         → List active rooms      |\n" +
		"|   • /exit          → Exit the program       |\n" +
		"|   • /help          → List of all commands   |\n" +
		"+---------------------------------------------+\033[0m\n" +
		"\033[32m💡 Enter your command:\033[0m \n"
)

func main() {
	// Membuat log untuk semua yang terjadi dalam server
	logF, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer logF.Close()
	logger = log.New(logF, "", log.LstdFlags)

	// Membuat koneksi
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Println("Failed to listen:", err)
		return
	}

	defer ln.Close()

	fmt.Println("Server started on :9090")

	// Goroutine untuk menyalurkan pesan ke anggota room
	go broadcaster()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("\033[31m\n❌ Failed to accept connection:\033[0m", err)
			continue
		}
		go handleConnection(conn)
	}
}

// Menangani koneksi satu client
func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)

	var name string

	// Phase memasukan nama yang unik
	for {
		conn.Write([]byte("\033[33m+-------------------------------------+\n"))
		conn.Write([]byte("|    🌐  Welcome to Terminal Chat!    |\n"))
		conn.Write([]byte("|  Where terminals come to life 💬    |\n"))
		conn.Write([]byte("+-------------------------------------+\033[0m\n\n"))
		conn.Write([]byte("\033[32m👤 Please enter your name:\033[0m \n")) // prompt tanpa \n agar input di baris yang sama

		// Menerima input nama dan mengecek nama sudah unik
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)

		// Menyimpan nama sudah valid(belum ada di server) atau tidak
		var valid string

		// Check di variable client
		for _, client := range clients {
			// Jika ada berarti tidak valid
			if client.name == name {
				valid = "taken"
				conn.Write([]byte("taken\n"))
				break
			}
		}

		// Jika valid maka keluar dari phase nama
		if valid != "taken" {
			conn.Write([]byte("ok\n"))
			break
		}

		// Pesan ke client kalau nama sudah ada yang punya
		conn.Write([]byte("\033[31m⚠️  Warning: username has been taken\033[0m\n"))
	}

	// Menyimpan client baru
	lock.Lock()
	client := &Client{name: name, conn: conn, incoming: make(chan string)}
	clients[conn] = client
	lock.Unlock()

	logger.Printf("%s connected from %s", client.name, conn.RemoteAddr())

	// Kirim pesan sambutan lobby
	lobbyMsg := fmt.Sprintf("\033[33m"+
		"\n+---------------------------------------------+\n"+
		"  👋 Welcome to the Lobby, %s!               \n"+
		helpMessage, client.name)

	conn.Write([]byte(lobbyMsg))

	// Goroutine untuk mengirim pesan ke client
	go sendMessages(client)

	// Membaca input berulang
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		input := scanner.Text()
		handleCommand(client, input, conn)
	}

	// Membersihkan saat disconect
	lock.Lock()
	delete(clients, conn)
	lock.Unlock()
	leaveRoom(client)
	conn.Close()
	logger.Printf("%s disconnected", client.name)
}

// Memproses perintah atau pesan umum
func handleCommand(client *Client, input string, conn net.Conn) {
	if strings.HasPrefix(input, "/") {

		switch {
		// Command join
		case strings.HasPrefix(input, "/join "):
			room := strings.TrimSpace(strings.TrimPrefix(input, "/join "))
			joinRoom(client, room)
		// Command leave the room
		case input == "/leave":
			if client.room == "" {
				conn.Write([]byte("\033[33m\nYou have not taken any Room\033[0m\n"))
				conn.Write([]byte(helpMessage))
			} else {
				leaveRoom(client)
				conn.Write([]byte("\033[33m\n+--------------------------------------------+\n"))
				conn.Write([]byte("| 🔔 You have left the room.                 |\n"))
				conn.Write([]byte("+--------------------------------------------+\n"))
				conn.Write([]byte(" 🏠 Welcome to Lobby, " + client.name + "!\n"))
				conn.Write([]byte(helpMessage))
			}

		// Command list room
		case input == "/rooms":
			listRooms(client)
		// Command keluar dari room
		case input == "/exit":
			client.conn.Close()
		// Case helps
		case input == "/help":
			client.incoming <- helpMessage

		// Command diluar yang sudah ada
		default:
			client.incoming <- "\033[31m❌ Unknown command.\033[0m\n\n\033[32m💡 Enter your command:\033[0m \n"
		}
	} else {

		if client.room == "" {
			client.incoming <- "\033[31m❌ Command not recognized. Please use a valid command.\033[0m\n\n\033[32m💡 Enter your command:\033[0m \n"

		} else {
			msg := Message{from: client.name, room: client.room, content: input}
			broadcast <- msg
			logger.Printf("[%s][%s]: %s", msg.room, msg.from, msg.content)
		}
	}
}

// Mengirim semua pesan yang masuk ke channel incoming client
func sendMessages(client *Client) {
	for msg := range client.incoming {
		client.conn.Write([]byte(msg))
	}
}

// Bergabung atau membuat room baru
func joinRoom(client *Client, room string) {
	leaveRoom(client)
	lock.Lock()
	rooms[room] = append(rooms[room], client)
	client.room = room
	lock.Unlock()
	client.incoming <- fmt.Sprintf("\033[34m\n+-------------------------------+\n"+
		"  🔗  Joined room: %-14s \n"+
		"+-------------------------------+\033[0m\n\n", room)
	broadcast <- Message{from: "\033[33mServer\033[0m", room: room, content: fmt.Sprintf("\033[33m>> %s has joined the room\033[0m", client.name)}
	logger.Printf("%s joined room '%s'", client.name, room)
}

func leaveRoom(client *Client) {
	// Memeriksa, apakah client dalam room ?
	//"" --> tidak di dalam room
	if client.room == "" {
		return
	}

	// Kunci akses data bersama
	lock.Lock()

	// Nama room
	roomName := client.room

	// Mengambil slice client dari room
	members := rooms[roomName]

	// Menghapus client dari room
	for i, c := range members {
		if c == client {
			rooms[roomName] = append(members[:i], members[i+1:]...)
			break
		}
	}

	// Memeriksa, apakah room jadi kosong?
	if len(rooms[roomName]) == 0 {
		delete(rooms, roomName)
		logger.Printf("Room '%s' is empty and has been deleted.", roomName)
	}

	// Update room client
	client.room = ""

	// Membuka lock
	lock.Unlock()

	// Broadcast cliet sudah keluar dari room
	broadcast <- Message{from: "\033[33mServer\033[0m", room: roomName, content: fmt.Sprintf("\033[33m>> %s has left the room\033[0m", client.name)}
	logger.Printf("%s left room '%s'", client.name, roomName)
}

// Menampilkan daftar room aktif
func listRooms(client *Client) {
	lock.Lock()
	defer lock.Unlock()
	if len(rooms) == 0 {
		client.incoming <- "\033[33m\nNo active rooms.\033[0m\n"
		return
	}

	client.incoming <- "\033[33m\n+--------------------------------+\n"
	client.incoming <- "| 📋 Active Room(s)              |\n"
	client.incoming <- "+--------------------------------+\n"
	for name, members := range rooms {
		client.incoming <- fmt.Sprintf("| - %-15s %2d user(s)   |\n", name, len(members))
	}
	client.incoming <- "+--------------------------------+\033[0m\n"
}

// Broadcast pesan ke setiap anggota room
func broadcaster() {
	for msg := range broadcast {
		lock.Lock()

		members := rooms[msg.room]
		for _, member := range members {
			if member.name != msg.from {
				member.incoming <- fmt.Sprintf("[%s]: %s\n", msg.from, msg.content)
			}

		}
		lock.Unlock()
	}
}
