//Server

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
	name     string 		//Nama wajib unik
	conn     net.Conn		//Koneksi jaringan dari server ke client
	incoming chan string	//Channel untuk mengantar pesan ke client
	room     string			//Room client berada
}

type Message struct {
	from    string			//Pengirim 
	room    string			//Tujuan room
	content string			//Isi pesan
}

//Mengelola state server
var (
	clients     = make(map[net.Conn]*Client)	//Semua client aktif di server
	rooms       = make(map[string][]*Client) 	//Semua nama room dan client di room tersebut
	broadcast   = make(chan Message)         	//Channel  pesan broadcast
	lock        = sync.Mutex{}					//Mutex untuk melindungi akses data bersama

	logger      *log.Logger		//Catat aktifitas 

	//Command yang dapat digunakan client, diawali dengan "/"
	lobby = "\033[33m" +
		"\n+---------------------------------------------+\n" +
		"|  🔧 Commands you can use:                   |\n" +
		"|   • /join <room>   → Join or create room    |\n" +
		"|   • /rooms         → List active rooms      |\n" +
		"|   • /leave         → Leave current room     |\n" +
		"|   • /exit          → Exit the program       |\n" +
		"|   • /help          → List of all commands   |\n" +
		"+---------------------------------------------+\033[0m\n" +
		"\033[32m💡 Enter your command:\033[0m \n"

	//Command bantuan untuk client
	helpMessage = "\033[33m" +
		"\n+---------------------------------------------+\n" +
		"|  🔧 Commands you can use:                   |\n" +
		"|   • /join <room>   → Join or create room    |\n" +
		"|   • /rooms         → List active rooms      |\n" +
		"|   • /leave         → Leave current room     |\n" +
		"|   • /exit          → Exit the program       |\n" +
		"|   • /help          → List of all commands   |\n" +
		"+---------------------------------------------+\033[0m\n" 
)

func main() {
	//Membuat log untuk semua aktivitas yang terjadi dalam server
	logFile, _ := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()
	logger = log.New(logFile, "", log.LstdFlags)

	//Membuat koneksi di port TCP 9090
	listener, err := net.Listen("tcp", ":9090")
	if err != nil { //Error jika port 9090 sudah digunakan oleh aplikasi lain 
		fmt.Println("\033[31m\n❌ Failed to listen:\033[0m", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server started on :9090")

	//Goroutine untuk menyalurkan pesan ke client (concurrency)
	go broadcaster()

	//Loop terus selama server aktif
	//Menerima jika ada client baru
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("\033[31m\n❌ Failed to accept connection:\033[0m", err)
			continue
		}
		go handleConnection(conn)
	}
}

//Menangani koneksi satu client
func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)

	//----INPUT NAMA CLIENT------
	var name string
	//Membuat nama client, harus unik
	//Loop hingga mendapat nama yang unik
	for {
		conn.Write([]byte("\033[33m+-------------------------------------+\n"))
		conn.Write([]byte("|    🌐  Welcome to Terminal Chat!    |\n"))
		conn.Write([]byte("|  Where terminals come to life 💬    |\n"))
		conn.Write([]byte("+-------------------------------------+\033[0m\n\n"))
		conn.Write([]byte("\033[32m👤 Please enter your name:\033[0m \n")) // prompt tanpa \n agar input di baris yang sama

		//Menerima input nama dan cek apakah unik
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)

		//Validasi nama sudah unik/belom
		var valid string
		for _, client := range clients {
			if client.name == name { //Jika nama udah terpakai
				valid = "name_taken"
				conn.Write([]byte("taken\n"))
				break
			}
		}

		//Jika valid, keluar dari loop
		if valid != "name_taken" {
			conn.Write([]byte("ok\n"))
			break
		}

		//Jika tidak valid, pesan ke client nama sudah ada yang punya
		conn.Write([]byte("\033[31m⚠️  Warning: username has been taken\033[0m\n"))
	}

	//----SIMPAN CLIENT & SAMBUTAN----
	//Menyimpan client baru
	lock.Lock()
	client := &Client{name: name, conn: conn, incoming: make(chan string)}
	clients[conn] = client
	lock.Unlock()

	logger.Printf("%s connected from %s", client.name, conn.RemoteAddr())

	//Kata sambutan
	lobbyMsg := fmt.Sprintf("\033[33m"+
		"\n+---------------------------------------------+\n"+
		"            👋 Welcome to Lobby, %s!              "+
		lobby, client.name)
	conn.Write([]byte(lobbyMsg))

	//Goroutine untuk mengirim pesan ke client ini
	go sendMessages(client)

	//---MEMBACA INPUT CLIENT---
	scanner := bufio.NewScanner(conn)
	//Loop terus menunggu input dari client
	for scanner.Scan() {
		input := scanner.Text()
		handleCommand(client, input, conn)
	}

	//CLIENT DISCONNECT
	lock.Lock()
	delete(clients, conn) 	//Hapus client
	lock.Unlock()
	leaveRoom(client)		//Otomatis keluar dari room
	conn.Close()			//Koneksi ke server putus
	logger.Printf("%s disconnected", client.name)
}

//Mengirim semua pesan masuk ke channel incoming client
func sendMessages(client *Client) {
	for msg := range client.incoming {
		client.conn.Write([]byte(msg))
	}
}

//Memproses input client
//Command: diawali "/"
//Pesan biasa
func handleCommand(client *Client, input string, conn net.Conn) {
	if strings.HasPrefix(input, "/") {  //Input command
		switch {
		case strings.HasPrefix(input, "/join "): //Command join
			room := strings.TrimSpace(strings.TrimPrefix(input, "/join "))
			joinRoom(client, room)
		case input == "/rooms": //Command list room
			listRooms(client)
		case input == "/leave": //Command leave the room
			if client.room == "" { //Client belum join room
				conn.Write([]byte("\033[33m\nYou have not taken any Room\033[0m\n"))
				conn.Write([]byte(lobby))
			} else {
				leaveRoom(client)
				conn.Write([]byte("\033[33m\n+--------------------------------------------+\n"))
				conn.Write([]byte("| 🔔 You have left the room.                 |\n"))
				conn.Write([]byte("+--------------------------------------------+\n"))
				conn.Write([]byte("            🏠 Welcome to Lobby, " + client.name + "!"))
				conn.Write([]byte(lobby))
			}
		case input == "/exit": //Command leave server
			client.conn.Close()
		case input == "/help": //Command help 
			client.incoming <- helpMessage
		default: //Command yang tidak ada di pilihan
			client.incoming <- "\033[31m❌ Unknown command.\033[0m\n\n\033[32m💡 Enter your command:\033[0m \n"
		}
	} else { //Input pesan biasa
		if client.room == "" { //Client belom join room
			client.incoming <- "\033[31m❌ Command not recognized. Please use a valid command.\033[0m\n\n\033[32m💡 Enter your command:\033[0m \n"
		} else {
			msg := Message{from: client.name, room: client.room, content: input}
			broadcast <- msg
			logger.Printf("[%s][%s]: %s", msg.room, msg.from, msg.content)
		}
	}
}

//Broadcast pesan ke setiap anggota di room
//Kirim pesan ke incoming client
func broadcaster() {
	for msg := range broadcast {
		lock.Lock()
		members := rooms[msg.room]
		for _, member := range members {
			if member.name != msg.from { //Agar pengirim tidak dikirimi pesan diri sendiri
				member.incoming <- fmt.Sprintf("[%s]: %s\n", msg.from, msg.content)
			}
		}
		lock.Unlock()
	}
}

//Bergabung / membuat room baru
func joinRoom(client *Client, room string) {
	leaveRoom(client) //Jika sebelumnya client dari room lain, otomatis keluar
	lock.Lock()
	rooms[room] = append(rooms[room], client) //Masukkin client ke room
	client.room = room
	lock.Unlock()
	client.incoming <- fmt.Sprintf("\033[34m\n+-------------------------------+\n"+
		"  🔗  Joined room: %-14s \n"+
		"+-------------------------------+\033[0m\n\n", room)
	broadcast <- Message{from: "\033[33mServer\033[0m", room: room, content: fmt.Sprintf("\033[33m>> %s has joined the room\033[0m", client.name)}
	logger.Printf("%s joined room '%s'", client.name, room)
}

func leaveRoom(client *Client) {
	if client.room == "" { //Belum join room
		return
	}

	lock.Lock()
	roomName := client.room
	members := rooms[roomName] //Ambil semua client dari room

	//Hapus client dari room
	for i, c := range members {
		if c == client {
			rooms[roomName] = append(members[:i], members[i+1:]...)
			break
		}
	}

	//Jika room jadi 0 client, hapus room
	if len(rooms[roomName]) == 0 {
		delete(rooms, roomName)
		logger.Printf("Room '%s' is empty and has been deleted.", roomName)
	}

	client.room = ""
	lock.Unlock()

	//Broadcast cliet sudah keluar dari room
	broadcast <- Message{from: "\033[33mServer\033[0m", room: roomName, content: fmt.Sprintf("\033[33m>> %s has left the room\033[0m", client.name)}
	logger.Printf("%s left room '%s'", client.name, roomName)
}

// Menampilkan daftar room aktif
func listRooms(client *Client) {
	lock.Lock()
	defer lock.Unlock()

	if len(rooms) == 0 { //Tidak ada room aktif
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