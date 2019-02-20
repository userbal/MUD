package main

import (
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"

	"golang.org/x/crypto/pbkdf2"
)

type Zone struct {
	ID    int
	Name  string
	Rooms []*Room
}

type Room struct {
	ID          int
	Zone        *Zone
	Name        string
	Description string
	Exits       [6]Exit
}

type Exit struct {
	To          *Room
	Description string
}

func readAllZones(db *sql.Tx) map[int]*Zone {
	var zones = make(map[int]*Zone)

	rows, err := db.Query("select id, name from zones ")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		z := new(Zone)
		//var id int
		//var name string
		//err = rows.Scan(&id, &name)
		err = rows.Scan(&z.ID, &z.Name)
		if err != nil {
			log.Fatal(err)
		}
		//z.ID = id
		//z.Name = name
		zones[z.ID] = z
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return zones
}

func readAllRooms(db *sql.Tx, zones map[int]*Zone) map[int]*Room {
	var rooms = make(map[int]*Room)

	rows, err := db.Query("select id, zone_id, name, description from rooms;")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		r := new(Room)
		//var id int
		var zone_id int
		//var name string
		//var description string
		err = rows.Scan(&r.ID, &zone_id, &r.Name, &r.Description)
		//err = rows.Scan(r.ID, r.Zone, r.Name, r.Description)
		if err != nil {
			log.Fatal(err)
		}
		//r.ID = id
		r.Zone = zones[zone_id]
		//r.Name = name
		//r.Description = description
		rooms[r.ID] = r
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return rooms
}

func readAllExits(db *sql.Tx, rooms map[int]*Room) {

	rows, err := db.Query("select * from exits;")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		e := new(Exit)
		var from_room_id int
		var to_room_id int
		var direction string
		//var description string
		//err = rows.Scan(from_room_id, &to_room_id, &direction, &description)
		err = rows.Scan(&from_room_id, &to_room_id, &direction, &e.Description)

		if err != nil {
			log.Fatal(err)
		}
		var x int
		if direction == "n" {
			x = 0
		}
		if direction == "e" {
			x = 1
		}
		if direction == "w" {
			x = 2
		}
		if direction == "s" {
			x = 3
		}
		if direction == "u" {
			x = 4
		}
		if direction == "d" {
			x = 5
		}
		e.To = rooms[to_room_id]
		//e.Description = description
		rooms[from_room_id].Exits[x] = *e
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func readPlayer(name string) *UandP {

	db, err := sql.Open("sqlite3", "world.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	UandP := new(UandP)

	row, err := tx.Query("select * from players where name = " + "\"" + name + "\"" + ";")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()
	var id int
	for row.Next() {
		err = row.Scan(&id, &UandP.name, &UandP.salt, &UandP.hash)

		if err != nil {
			log.Fatal(err)
		}
	}
	err = row.Err()
	if err != nil {
		log.Fatal(err)
	}

	if UandP.name == "" {
		UandP.password = "error"
	}

	tx.Commit()
	return UandP
}

func createNewUser(UandP *UandP) *UandP {

	salt := make([]byte, 32)
	_, err := crand.Read(salt)
	salt64 := base64.StdEncoding.EncodeToString(salt)
	UandP.salt = salt64

	UandP.hash = pbkdf2.Key(
		[]byte(UandP.password),
		salt,
		64*1024,
		32,
		sha256.New)

	UandP.password = string(UandP.hash)

	fmt.Println("\nsalt:")
	fmt.Println(UandP.salt)

	fmt.Println("\npassword:")
	fmt.Println(UandP.password)

	db, err := sql.Open("sqlite3", "world.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.Exec("insert into players(name, salt, hash) values (" + "\"" + UandP.name + "\"" + "," + "\"" + UandP.salt + "\"" + "," + "\"" + UandP.password + "\"" + ");")
	if err != nil {
		log.Fatal(err)
	}

	tx.Commit()
	return UandP
}
