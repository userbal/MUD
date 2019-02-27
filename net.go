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

type eventIN struct {
	query      []string
	player     *Player
	connBroken bool
}

type UandP struct {
	name     string
	hash     []byte
	password string
	salt     string
}

func compareUsernameAndPasswordToDatabase(submission *UandP) string {
	_, ok := Players[submission.name]
	if ok {
		return "player already logged in"
	}

	record := readPlayer(submission.name)

	if record.password == "error" {
		return "Username not found"
	} else {
		salt, err := base64.StdEncoding.DecodeString(record.salt)
		if err != nil {
			log.Printf("listen failed: %v", err)
		}

		submission.hash = pbkdf2.Key(
			[]byte(submission.password),
			salt,
			64*1024,
			32,
			sha256.New)

		if subtle.ConstantTimeCompare(record.hash, submission.hash) != 1 {
			log.Printf("incorrect password, username: %v", submission.name)
			return "passwords do not match"
		} else {
			return "passwords match"
		}
	}
}

func loginUser(conn net.Conn) *Player {
	player := new(Player)
	a := getUsernameAndPassword(conn)
	b := compareUsernameAndPasswordToDatabase(a)
	switch b {
	case "passwords match":
		player = initializePlayer(a, conn)
		log.Printf("Player logged in, username: %v", player.name)
		return player
	case "passwords do not match":

		fmt.Fprintf(conn, "incorrect password\n")
	case "Username not found":
		fmt.Fprintln(conn, "creating new user")
		createNewUser(a)
	case "player already logged in":
		fmt.Fprintln(conn, "Sorry, that player is already logged in")

	}
	player = new(Player)
	return player
}

func initializePlayer(a *UandP, conn net.Conn) *Player {
	player := new(Player)
	player.name = a.name
	player.current_room = Rooms[3001]
	Players[player.name] = player
	Rooms[3001].Players[player.name] = player
	tellPlayersPlayerEntered(3001)
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
	if err := scanner.Err(); err != nil {
		log.Printf("scanner: %v", err)
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
	if err := scanner.Err(); err != nil {
		log.Printf("scanner: %v", err)
	}

	return UandP
}

func connectionStarter(InCh chan eventIN) {

	//watch for connections
	fmt.Printf("Listening\n")

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Printf("listen failed: %v", err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("opening conn error (func connectionStarter) : %v", err)
			continue
		}

		player := loginUser(conn)
		if player.name != "" {

			go playerIn(player.conn, player, InCh)
			go playerOUT(player.conn, player)
			log.Printf("go routines created for %v", player.name)

		} else {
			conn.Close()
		}

	}
}

func playerIn(conn net.Conn, player *Player, ch chan eventIN) {
	loggedOut := false
	//call look so the player sees something when they log in
	event := new(eventIN)
	var initquery []string
	initquery = append(initquery, "look", "")
	event.query = initquery
	event.player = player
	event.connBroken = false
	ch <- *event

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {

		var query []string

		line := strings.Fields(scanner.Text())
		//if they don't send anything don't send anything throught the channel
		if len(line) < 1 {
			continue
		}

		//fill the slice of strings with the seperated strings
		query = append(query, line[0], strings.Join(line[1:], " "))
		if line[0] == "quit" {
			loggedOut = true
			break
		}
		event.query = query
		event.player = player
		event.connBroken = false
		ch <- *event
	}

	if err := scanner.Err(); err != nil {
		log.Printf("scanner error (func playerIn): %v", err)
	}

	if loggedOut {
		log.Printf("Player logged out, username: %v", player.name)
	} else {
		log.Printf("Player disconnected without logging out, username: %v", player.name)
	}

	event.player = player
	event.connBroken = true
	ch <- *event
	log.Printf("Quit process - goIN closed, username: %v", player.name)
}

func playerOUT(conn net.Conn, player *Player) {
	defer conn.Close()

	for message := range player.channel {
		response := message.response
		fmt.Fprintf(conn, response)

	}
	log.Printf("Quit process - connection closed, username: %v", player.name)
	log.Printf("Quit process - goOUT and connection closed, username: %v", player.name)
	return
}
