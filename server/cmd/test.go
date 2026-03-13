package main

import (
	"fmt"
	"log"

	"github.com/sdomino/scribble"
)

type Fish struct {
	Name string
}

func main() {
	dir := "./filedb"
	db, err := scribble.New(dir, nil)
	if err != nil {
		log.Fatal(err)
	}
	fish := &Fish{}
	if err := db.Read("fish", "onefish", fish); err != nil {
		fmt.Println("Error", err)
	}
	fmt.Println(fish.Name)

}
