package main

import (
	"bufio"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

type UandP struct {
	name     string
	hash     []byte
	password string
	salt     string
}

func compareUsernameAndPasswordToDatabase(submission *UandP) string {
	record := readPlayer(submission.name)

	if record.password == "error" {
		return "Username not found"
	} else {
		salt, err := base64.StdEncoding.DecodeString(record.salt)
		if err != nil {
			log.Fatalf("listen failed: %v", err)
		}

		submission.hash = pbkdf2.Key(
			[]byte(submission.password),
			salt,
			64*1024,
			32,
			sha256.New)

		if subtle.ConstantTimeCompare(record.hash, submission.hash) != 1 {
			return "passwords do not match"
		} else {
			return "passwords match"
		}
	}
}

func loginUser(conn net.Conn) *Player {
	a := getUsernameAndPassword(conn)
	b := compareUsernameAndPasswordToDatabase(a)
	switch b {
	case "passwords match":
		player := initializePlayer(a, conn)
		return player
	case "passwords do not match":
		fmt.Fprintf(conn, "incorrect password")
		loginUser(conn)
	case "Username not found":
		fmt.Fprintln(conn, "creating new user")
		createNewUser(a)
		loginUser(conn)
	}
	player := initializePlayer(a, conn)
	return player
}

func initializePlayer(a *UandP, conn net.Conn) *Player {
	player := new(Player)
	player.name = a.name
	player.current_room = Rooms[3001]
	player.conn = conn
	player.channel = make(chan eventOUT)
	return player
}

func getUsernameAndPassword(conn net.Conn) *UandP {

	scanner := bufio.NewScanner(conn)
	UandP := new(UandP)

	fmt.Fprintf(conn, "\n\nName: ")
	for scanner.Scan() {
		line := scanner.Text()

		//if line has isn't empty, send it throught the channel
		if len(line) > 0 {
			UandP.name = line
			break
		}
	}

	fmt.Fprintf(conn, "Password: ")
	for scanner.Scan() {
		line := scanner.Text()

		//if line has isn't empty, send it throught the channel
		if len(line) > 0 {
			UandP.password = line
			break
		}
	}

	return UandP
}

func connectionStarter(InCh chan eventIN) {

	//watch for connections
	fmt.Printf("Listening\n")

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		player := loginUser(conn)
		go playerIn(conn, player, InCh)
		go playerOUT(conn, player)

	}
}

func playerIn(conn net.Conn, player *Player, ch chan eventIN) {

	fmt.Fprintf(conn, "\n> ")
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var query []string
		event := new(eventIN)

		//create an event struct and a slice of strings to carry the query

		// recieve the scanner text, seperate it into chunks, and save it into line
		line := strings.Fields(scanner.Text())

		//if line has isn't empty, send it throught the channel
		if len(line) > 0 {

			//fill the slice of strings with the seperated strings
			query = append(query, line[0], strings.Join(line[1:], " "))

			event.query = query
			event.player = player
			ch <- *event
			fmt.Fprintf(conn, "\n> ")
		}

	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanner: %v", err)
		var query []string
		event := new(eventIN)
		query = append(query, "connection closed by client")
		event.query = query
		event.player = player
		ch <- *event
	}
}

func playerOUT(conn net.Conn, player *Player) {
	for x := range player.channel {
		message := x
		response := message.response
		fmt.Fprintf(conn, response)
	}
	conn.Close()
	return
}

//func main() {
//
//InCh := make(chan event)
//go connectionStarter(InCh)
//for {
////InCh := callCommand(verb, stringl, commands, player)
//fmt.Println(<-InCh)
//}
//
//}
