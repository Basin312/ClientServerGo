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
	clients   = make(map[net.Conn]*Client) //list client yang ada di server
	rooms     = make(map[string][]*Client) //list room dan client yang ada di room tersebut
	broadcast = make(chan Message)
	lock      = sync.Mutex{}
	logger    *log.Logger
)

func main() {
	// bikin log untuk semua yang terjadi dalam server
	logF, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer logF.Close()
	logger = log.New(logF, "", log.LstdFlags)

	// bikin koneksi
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Println("Failed to listen:", err)
		return
	}

	defer ln.Close()

	fmt.Println("Server started on :9090")

	go broadcaster()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)

	var name string

	// phase memasukan nama
	for {
		conn.Write([]byte("\033[33m+-------------------------------------+\n"))
		conn.Write([]byte("|    🌐  Welcome to Terminal Chat!    |\n"))
		conn.Write([]byte("|  Where terminals come to life 💬    |\n"))
		conn.Write([]byte("+-------------------------------------+\033[0m\n\n"))
		conn.Write([]byte("\033[32m👤 Please enter your name:\033[0m \n")) // prompt tanpa \n agar input di baris yang sama

		//menerima input nama
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)

		// untuk menyimpan namanya valid(belum ada di server) atau tidak
		var valid string

		// check di di variable client
		for _, client := range clients {
			// kalau ada berarti tidak valid
			if client.name == name {
				valid = "taken"
				conn.Write([]byte("taken\n"))
				break
			}
		}

		// jika valid maka keluar dari phase nama
		if valid != "taken" {
			conn.Write([]byte("ok\n"))
			break
		}

		// message ke client kalau nama sudah ada punya
		conn.Write([]byte("Warning: username has been taken\n"))
	}

	lock.Lock()
	client := &Client{name: name, conn: conn, incoming: make(chan string)}
	clients[conn] = client
	lock.Unlock()

	logger.Printf("%s connected from %s", client.name, conn.RemoteAddr())


	lobbyMsg := fmt.Sprintf("\033[33m" +
    "\n+---------------------------------------------+\n" +
    "  👋 Welcome to the Lobby, %s!               \n" +
    "|  🔧 Commands you can use:                   |\n" +
    "|   • /join <room>  - join a chat room        |\n" +
    "|   • /leave       - leave current room       |\n" +
    "|   • /rooms       - list available rooms     |\n" +
    "|   • /exit        - exit the program         |\n" +
    "+---------------------------------------------+\033[0m\n" +
	"\033[32m💡 Enter your command:\033[0m \n", client.name)

	conn.Write([]byte(lobbyMsg))

	go sendMessages(client)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		input := scanner.Text()
		handleCommand(client, input, conn)
	}

	lock.Lock()
	delete(clients, conn)
	lock.Unlock()
	leaveRoom(client)
	conn.Close()
	logger.Printf("%s disconnected", client.name)
}

func handleCommand(client *Client, input string, conn net.Conn) {
	if strings.HasPrefix(input, "/") {

		switch {
		// command join
		case strings.HasPrefix(input, "/join "):
			room := strings.TrimSpace(strings.TrimPrefix(input, "/join "))
			joinRoom(client, room)
		//command leave the room
		case input == "/leave":
			// check sudah join room sebelumnya atau tidak
			if client.room == "" {
				conn.Write([]byte("You have not taken any Room\n"))
			} else {
				leaveRoom(client)
				conn.Write([]byte("You have leave the Room\n"))
				conn.Write([]byte("Welcome to Lobby, " + client.name + "! \n Use:\n /join <room> : client can join to any room \n /leave: client can leave the room(if only in the) \n /exit: client close the program,\n /rooms: client can see the room that exist  \n"))
			}
		// command list room
		case input == "/rooms":
			listRooms(client)
		// command keluar dari room
		case input == "/exit":
			client.conn.Close()
		//command diluar yang sudah ada
		default:
			client.incoming <- "Unknown command.\n"
		}
	} else {

		if client.room == "" {
			client.incoming <- "\033[31m\n❌ Command not recognized. Please use a valid command.\033[0m\n\033[32m💡 Enter your command:\033[0m \n"

		} else {
			msg := Message{from: client.name, room: client.room, content: input}
			broadcast <- msg
			logger.Printf("[%s][%s]: %s", msg.room, msg.from, msg.content)
		}
	}
}

func sendMessages(client *Client) {
	for msg := range client.incoming {
		client.conn.Write([]byte(msg))
	}
}

func joinRoom(client *Client, room string) {
	leaveRoom(client)
	lock.Lock()
	rooms[room] = append(rooms[room], client)
	client.room = room
	lock.Unlock()
	client.incoming <- fmt.Sprintf("You joined room '%s'\n", room)
	broadcast <- Message{from: "Server", room: room, content: fmt.Sprintf(">> %s has joined the room\n", client.name)}
	logger.Printf("%s joined room '%s'", client.name, room)
}

func leaveRoom(client *Client) {
	if client.room == "" {
		return
	}
	lock.Lock()
	members := rooms[client.room]
	for i, c := range members {
		if c == client {
			rooms[client.room] = append(members[:i], members[i+1:]...)
			break
		}
	}
	roomName := client.room
	client.room = ""
	lock.Unlock()
	broadcast <- Message{from: "Server", room: roomName, content: fmt.Sprintf(">> %s has left the room\n", client.name)}
	logger.Printf("%s left room '%s'", client.name, roomName)
}

func listRooms(client *Client) {
	lock.Lock()
	defer lock.Unlock()
	if len(rooms) == 0 {
		client.incoming <- "No active rooms.\n"
		return
	}
	client.incoming <- "Active rooms:\n"
	for name, members := range rooms {
		client.incoming <- fmt.Sprintf("- %s (%d user(s))\n", name, len(members))
	}
}

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
