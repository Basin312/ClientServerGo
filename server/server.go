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
	clients   = make(map[net.Conn]*Client)
	rooms     = make(map[string][]*Client)
	broadcast = make(chan Message)
	lock      = sync.Mutex{}
	logger    *log.Logger
)

func main() {
	logF, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer logF.Close()
	logger = log.New(logF, "", log.LstdFlags)

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
	conn.Write([]byte("Enter your username:\n"))
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	lock.Lock()
	for _, client := range clients {
		if client.name == name {
			conn.Write([]byte("Username already taken.\n"))
			conn.Close()
			lock.Unlock()
			return
		}
	}
	client := &Client{name: name, conn: conn, incoming: make(chan string)}
	clients[conn] = client
	lock.Unlock()

	logger.Printf("%s connected from %s", client.name, conn.RemoteAddr())
	conn.Write([]byte("Welcome, " + client.name + "! Use /join <room>, /leave, /exit, /rooms\n"))

	go sendMessages(client)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		input := scanner.Text()
		handleCommand(client, input)
	}

	lock.Lock()
	delete(clients, conn)
	lock.Unlock()
	leaveRoom(client)
	conn.Close()
	logger.Printf("%s disconnected", client.name)
}

func handleCommand(client *Client, input string) {
	if strings.HasPrefix(input, "/") {
		switch {
		case strings.HasPrefix(input, "/join "):
			room := strings.TrimSpace(strings.TrimPrefix(input, "/join "))
			joinRoom(client, room)
		case input == "/leave":
			leaveRoom(client)
		case input == "/rooms":
			listRooms(client)
		case input == "/exit":
			client.conn.Close()
		default:
			client.incoming <- "Unknown command.\n"
		}
	} else {
		if client.room == "" {
			client.incoming <- "Join a room to send messages.\n"
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
