package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type (
	Player struct {
	}
)

func (p *Player) Run(globals *Globals) error {

	files, err := os.ReadDir("players")
	if err != nil {
		return err
	}
	for _, file := range files {
		contents, err := os.ReadFile(fmt.Sprintf("players/%s", file.Name()))
		if err != nil {
			return err
		}

		reader := strings.NewReader(string(contents))
		tokenizer := html.NewTokenizer(reader)
		count := 0
		var name, team, cost, games, points string
		for {
			tt := tokenizer.Next()
			if tt == html.ErrorToken {
				if tokenizer.Err() == io.EOF {
					return errors.New("tokenizer EOF error")
				}
				fmt.Printf("Error: %v", tokenizer.Err())
				return errors.New("tokenizer error")
			}
			count++

			if count > 70 {
				break
			}
			if count == 51 {
				name = strings.Trim(tokenizer.Token().Data, " ")
			}
			if count == 55 {
				team = tokenizer.Token().Data
			}
			if count == 59 {
				cost = tokenizer.Token().Data
			}
			if count == 63 {
				games = tokenizer.Token().Data
			}
			if count == 67 {
				points = tokenizer.Token().Data
				fmt.Printf("%s,%s,%s,%s,%s\n", name, team, cost, games, points)
			}
		}
	}
	return nil
}
