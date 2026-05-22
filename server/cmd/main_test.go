package main

import (
	"log"
	"strings"
	"testing"
)

type Fish struct {
	Name string
}

func cleanInput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func TestB(t *testing.T) {
	data := "    dfdf  \r\n dfdf   \rdd  \n33  "

	log.Printf("原始: %q\n", data)
	log.Printf("处理后: %q\n", cleanInput(data))
}

//func TestA(t *testing.T) {
//	dir := "./filedb"
//	db, err := scribble.New(dir, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fish := &Fish{}
//	if err := db.Read("fish", "onefish", fish); err != nil {
//		fmt.Println("Error", err)
//	}
//	fmt.Println(fish.Name)
//}
