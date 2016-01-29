package main

import (
    "github.com/julianshen/gonude"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

func main() {
    if len(os.Args) < 2 {
        log.Println("Usage: ", os.Args[0], "[filename]")
        os.Exit(0)
    }
	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0777)
	defer file.Close()

	if err != nil {
		log.Println(err)
	} else {
		img, itype, err := image.Decode(file)
		log.Println("type: " + itype)
		if err != nil {
			log.Println(err)
		} else {
			log.Println(gonude.IsNude(&img))
		}
	}
}
