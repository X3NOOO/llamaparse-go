package main

import (
	"fmt"
	"os"

	"github.com/X3NOOO/llamaparse-go"
)

const FILENAME = "somatosensory.pdf"

func main() {
	file, _ := os.ReadFile(FILENAME)

	parsedText, err := llamaparse.Parse(file, llamaparse.MARKDOWN, nil, nil, nil, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(parsedText)
}