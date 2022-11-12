package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/chromedp/chromedp"
)

var baseUrl = "https://www.chessgames.com"
var gameRegex = regexp.MustCompile(`pgn=\"((.|\n)+)\" ratio`)

func GetGame(url string) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var body string

	err := chromedp.Run(ctx,
		chromedp.Navigatgit e(url),
		chromedp.OuterHTML("#olga-data", &body, chromedp.ByQuery),
	)
	if err != nil {
		return "", err
	}

	pgn := gameRegex.FindAllStringSubmatch(body, -1)

	if len(pgn) == 0 {
		return "", errors.New("No PGNs found")
	}

	if len(pgn[0]) < 2 {
		return "", errors.New("No matching PGN data at game url")
	}

	return html.UnescapeString(pgn[0][1]), nil
}

func GetCollection(url string) ([]string, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalf("Failed to find game collection")
	}

	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)
	body := string(bytes)

	m := regexp.MustCompile(`\/perl\/chessgame\?gid=\d{4,}`) // structure of game reference
	games := m.FindAllString(body, -1)                       // pull out the list of games

	for i, g := range games {
		games[i] = baseUrl + g
	}

	return games, nil
}

func FetchAndWriteGames(games []string, fileName string) {
	// create/truncate file to write to
	f, err := os.Create(fileName)

	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	numGames := len(games)
	for i, g := range games {
		fmt.Println(baseUrl + g)
		game, err := GetGame(baseUrl + g)

		fmt.Println(game)
		if err != nil {
			log.Printf("Skipping - Failed to download %s because %s", g, err.Error())
		} else {
			f.WriteString(game)

			// put spacing if multiple games
			if i < numGames-1 {
				f.WriteString("\n\n")
			}
		}
	}
}

func main() {
	urlPtr := flag.String("url", "", "URL for game or game collection")
	pgnPtr := flag.String("pgn", "", "PGN filename to write results to")

	flag.Parse()
	url := *urlPtr

	var games []string
	var err error

	if url == "" {
		log.Fatal("Invalid URL (it's empty or unprovided)")
	}

	if *pgnPtr == "" {
		log.Fatal("Invalid output PGN name (it's empty or unprovided)")
	}

	if strings.Contains(url, "chesscollection") {
		games, err = GetCollection(url)
	} else if strings.Contains(url, "chessgame") {
		games = []string{url}
	} else {
		err = errors.New("Invalid collection or game URL")
	}

	if err != nil {
		log.Fatal(err)
	} else {
		FetchAndWriteGames(games, *pgnPtr)
		fmt.Printf("Wrote %d games from %s to file %s\n", len(games), url, *pgnPtr)
	}
}
