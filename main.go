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
	"strconv"
	"strings"

	"github.com/chromedp/chromedp"
)

var baseUrl = "https://www.chessgames.com"
var gameRegex = regexp.MustCompile(`pgn=\"((.|\n)+)\" ratio`)
var gameUrlRegex = regexp.MustCompile(`\/perl\/chessgame\?gid=\d{4,}`) // structure of game reference
var totalWritten = 0
var ctx context.Context
var cancel context.CancelFunc

func GetGame(url string) (string, error) {
	var body string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
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

/*
  - Gives the number of pages in a chessgames page.
    Returns 1 if not a multi-page collection
*/
func PageCount(body string) int {
	p := regexp.MustCompile(`page \d+ of (\d+); games`)
	matches := p.FindStringSubmatch(body)

	if matches == nil {
		return 1
	} else if matches[1] != "" {
		if conv, err := strconv.Atoi(matches[1]); err != nil {
			return 1
		} else {
			return conv
		}
	}

	return 1
}

func GetCollectionSinglePage(url string, currBody *string) ([]string, error) {
	log.Println("Going to single page: ", url)
	var body string

	if currBody == nil {
		resp, err := http.Get(url)

		if err != nil {
			log.Fatalf("Failed to find game collection")
		}

		defer resp.Body.Close()
		bytes, _ := io.ReadAll(resp.Body)
		body = string(bytes)
	} else {
		body = *currBody
	}

	games := gameUrlRegex.FindAllString(body, -1) // pull out the list of games

	for i, g := range games {
		games[i] = baseUrl + g
	}

	return games, nil
}

func GetCollection(url string) ([]string, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalf("Failed to find game collection")
	}

	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)
	body := string(bytes)

	pageCount := PageCount(body)

	if pageCount == 1 {
		return GetCollectionSinglePage(url, &body)
	} else {
		// recreate URL without page number
		// then add it for each page
		pagePattern := regexp.MustCompile(`&?page=\d+`)
		pageLess := pagePattern.ReplaceAllLiteralString(url, "")

		var games []string
		for page := 1; page <= pageCount; page++ {
			withPage := fmt.Sprintf("%s&page=%d", pageLess, page)
			gameCol, err := GetCollectionSinglePage(withPage, nil)

			if err != nil {
				log.Printf("Had error getting page: %s due to error %v, but continuing", withPage, err)
			}
			games = append(games, gameCol...)
		}

		return games, nil
	}
}

func DedupGameList(games []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, entry := range games {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}

func FetchAndWriteGames(games []string, fileName string) {
	// create/truncate file to write to
	f, err := os.Create(fileName)

	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	games = DedupGameList(games)
	numGames := len(games)

	for i, g := range games {
		game, err := GetGame(g)

		if err != nil {
			log.Printf("Skipping game %d - Failed to download %s because %s", i+1, g, err.Error())
		} else {
			log.Printf("Writing game %d - %s", i+1, g)
			totalWritten++
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

	if strings.Contains(url, "/chessgame?") {
		games = []string{url}
	} else if strings.Contains(url, "/chesscollection?") {
		games, err = GetCollection(url)
	} else { // TODO: support other endpoints, but for now, treat them like chesscollections
		log.Println("Not a chess collection, but still searching endpoint for chessgames...")

		games, err = GetCollection(url)
	}

	if err != nil {
		log.Fatal(err)
	} else {
		ctx, cancel = chromedp.NewContext(context.Background())
		defer cancel()
		FetchAndWriteGames(games, *pgnPtr)

		log.Printf("Wrote %d games from %s to file %s\n", totalWritten, url, *pgnPtr)
	}
}
