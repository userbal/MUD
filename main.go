package main

import (
	"database/sql"
	"log"
	"net"

	_ "github.com/mattn/go-sqlite3"
)

var Zones map[int]*Zone
var Rooms map[int]*Room
var Users map[string]string

var commands map[string]func(string, *Player) string

func addCommand(command string, action func(string, *Player) string, commands map[string]func(string, *Player) string) {
	commands[command] = action
}

type Player struct {
	name         string
	current_room *Room
	conn         net.Conn
	channel      chan eventOUT
}

type eventIN struct {
	query  []string
	player *Player
}

type eventOUT struct {
	response string
}

var compass = map[string]int{
	"north": 0,
	"east":  1,
	"west":  2,
	"south": 3,
	"up":    4,
	"down":  5,
}

var Rcompass = map[int]string{
	0: "n",
	1: "e",
	2: "w",
	3: "s",
	4: "u",
	5: "d",
}

func doLook(arg string, player *Player) string {
	response := ""
	if arg == "" {
		response += (player.current_room.Name + "\n\n")
		response += (player.current_room.Description + "\n")
		response += "[ Exits:  "
		for i := 0; i < 6; i++ {
			if player.current_room.Exits[i].Description != "" {
				response += Rcompass[i] + "  "
			}
		}

		response += "]\n"
	} else if val, ok := compass[arg]; ok {
		response = player.current_room.Exits[val].Description
	} else {
		response = arg + " invalid\n"
	}
	return response
}

func doMove(arg string, player *Player) string {
	response := ""
	if val, ok := compass[arg]; ok {
		if player.current_room.Exits[val].To != nil {
			player.current_room = player.current_room.Exits[val].To
			response = player.current_room.Description
		} else {
			response = "can't go there"
		}
	} else {
		response = arg + " invalid"
	}
	return response
}

func doRecall(arg string, player *Player) string {
	response := ""
	player.current_room = Rooms[3001]
	response += player.current_room.Description
	return response
}

func quit(arg string, player *Player) string {
	close(player.channel)
	player.channel = nil
	return "closed succesfully"
}

func callCommand(verb string, line string, commands map[string]func(string, *Player) string, player *Player) string {
	response := ""
	//var command func(string)
	//var ok bool
	//command, ok = commands[verb]
	if command, ok := commands[verb]; ok {
		response = command(line, player)
	} else {
		response = "Not a Command\n"
	}
	return response
}

func readDB() {

	db, err := sql.Open("sqlite3", "world.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	zones := readAllZones(tx)
	if err != nil {
		log.Fatal(err)
	}

	tx.Commit()

	tx, err = db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	Rooms = readAllRooms(tx, zones)
	if err != nil {
		log.Fatal(err)
	}

	//readRoom(tx)
	tx.Commit()

	tx, err = db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	readAllExits(tx, Rooms)
	if err != nil {
		log.Fatal(err)
	}
	//readRoom(tx)
	tx.Commit()

}

func main() {

	readDB()
	Users = make(map[string]string)
	commands := make(map[string]func(string, *Player) string)
	addCommand("look", doLook, commands)
	addCommand("recall", doRecall, commands)
	addCommand("move", doMove, commands)
	addCommand("quit", quit, commands)
	InCh := make(chan eventIN)
	go connectionStarter(InCh)
	eventIn := <-InCh
	for {
		if eventIn.player.channel != nil {

			//var response string
			eventout := new(eventOUT)
			eventout.response = "\n" + callCommand(eventIn.query[0], eventIn.query[1], commands, eventIn.player) + "\n> "
			eventIn.player.channel <- *eventout
		}
	}

}
