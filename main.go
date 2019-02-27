package main

import (
	"log"
	"net"
)

var Zones map[int]*Zone
var Rooms map[int]*Room
var Players = map[string]*Player{}

type Player struct {
	name         string
	current_room *Room
	conn         net.Conn
	channel      chan eventOUT
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
	"n":     0,
	"e":     1,
	"w":     2,
	"s":     3,
	"u":     4,
	"d":     5,
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

		//add players in room with you
		response += "[ Players: "
		for i := range Rooms[player.current_room.ID].Players {
			if i == player.name {
				continue
			}
			response += i + ", "
		}
		response += "]\n"
	} else if val, ok := compass[arg]; ok {
		response = player.current_room.Exits[val].Description
	} else {
		response = arg + " invalid\n"
	}
	return response
}

func doSay(arg string, player *Player) string {
	response := ""
	if arg == "" {
		response += "\"say\" syntax: say whatever you want to say"
		return response
	} else {
		for name, playerPointer := range Rooms[player.current_room.ID].Players {
			if name != player.name {
				eventout := new(eventOUT)
				eventout.response = "\n" + player.name + " says: " + arg + "\n\n"
				playerPointer.channel <- *eventout
			}
		}
	}
	response = "You said: " + arg + "\n"
	return response
}

func doGossip(arg string, player *Player) string {
	response := ""
	if arg == "" {
		response += "\"gossip\" syntax: gossip whatever you want to say"
		return response
	} else {
		for name, playerPointer := range Players {
			if name != player.name {
				eventout := new(eventOUT)
				eventout.response = "\n" + player.name + " says: " + arg + "\n\n"
				playerPointer.channel <- *eventout
			}
		}
	}
	response = "You said: " + arg + "\n"
	return response
}

func doMove(arg string, player *Player) string {
	response := ""
	if val, ok := compass[arg]; ok {
		if player.current_room.Exits[val].To != nil {
			//change the players location:
			//1. remove the player from the current room
			tellPlayersPlayerExited(player.current_room.ID)
			delete(Rooms[player.current_room.ID].Players, player.name)
			//2. notify the player
			player.current_room = player.current_room.Exits[val].To
			//3. let the room know the player is there
			Rooms[player.current_room.ID].Players[player.name] = player
			tellPlayersPlayerEntered(player.current_room.ID)
			response = player.current_room.Description

			response += "[ Exits:  "
			for i := 0; i < 6; i++ {
				if player.current_room.Exits[i].Description != "" {
					response += Rcompass[i] + "  "
				}
			}
			response += "]\n"

			//add players in room with you
			response += "[ players: "
			for i := range Rooms[player.current_room.ID].Players {
				if i != player.name {
					response += i + ", "
				}
			}
			response += "]\n"
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
	tellPlayersPlayerExited(player.current_room.ID)
	player.current_room = Rooms[3001]
	response += player.current_room.Name + "\n\n"
	response += player.current_room.Description + "\n"
	//change the players location:
	//1. remove the player from the current room
	tellPlayersPlayerEntered(player.current_room.ID)
	delete(Rooms[player.current_room.ID].Players, player.name)
	//2. let the room know the player is there
	Rooms[player.current_room.ID].Players[player.name] = player

	//exits
	response += "[ Exits:  "
	for i := 0; i < 6; i++ {
		if player.current_room.Exits[i].Description != "" {
			response += Rcompass[i] + "  "
		}
	}
	response += "]\n"

	//add players in room with you
	response += "[ Players: "
	for i := range Rooms[player.current_room.ID].Players {
		if i != player.name {
			response += i + ", "
		}
	}
	response += "]\n"
	return response
}

func doQuit(arg string, player *Player) string {
	log.Printf("Player quit, username: %v", player.name)
	close(player.channel)
	player.channel = nil
	delete(Players, player.name)
	delete(Rooms[player.current_room.ID].Players, player.name)
	log.Printf("Quit process - Player removed from rooms, world, username: %v", player.name)
	player = nil
	return ""
}

var commands map[string]func(string, *Player) string

func addCommand(command string, action func(string, *Player) string, commands map[string]func(string, *Player) string) {
	commands[command] = action
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

func tellPlayersPlayerEntered(roomID int) {

	for name := range Rooms[roomID].Players {
		response := ""
		eventout := new(eventOUT)
		response += "\n" + name
		for otherName, otherPlayerPointer := range Rooms[roomID].Players {
			if name != otherName {
				response += " entered the room.\n\n"
				eventout.response = response
				if otherPlayerPointer.channel != nil {
					otherPlayerPointer.channel <- *eventout
				}
			}
		}
	}
}

func tellPlayersPlayerExited(roomID int) {

	for name := range Rooms[roomID].Players {
		response := ""
		eventout := new(eventOUT)
		response += name
		for otherName, otherPlayerPointer := range Rooms[roomID].Players {
			if name != otherName {
				response += " exited the room.\n\n"
				eventout.response = response
				if otherPlayerPointer.channel != nil {
					otherPlayerPointer.channel <- *eventout
				}
			}
		}
	}
}

func main() {

	readDB()

	commands := make(map[string]func(string, *Player) string)
	addCommand("look", doLook, commands)
	addCommand("recall", doRecall, commands)
	addCommand("move", doMove, commands)
	addCommand("quit", doQuit, commands)
	addCommand("say", doSay, commands)
	addCommand("gossip", doGossip, commands)

	InCh := make(chan eventIN)
	go connectionStarter(InCh)
	for eventIn := range InCh {

		if eventIn.connBroken {
			doQuit("", eventIn.player)
		}

		if eventIn.player.channel == nil {
			continue
		}
		eventout := new(eventOUT)
		response := callCommand(eventIn.query[0], eventIn.query[1], commands, eventIn.player)
		eventout.response = "\n" + response + "\n"
		eventIn.player.channel <- *eventout
	}

}
