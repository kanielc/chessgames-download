package main

import (
	"context"
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

	"github.com/PuerkitoBio/goquery"
)

var baseUrl = "https://www.chessgames.com"
var gameUrlRegex = regexp.MustCompile(`\/perl\/chessgame\?gid=\d{4,}`) // structure of game reference
var whiteMoves = regexp.MustCompile(`([0-9]+\. [a-zA-Z-]+[0-9]?[a-zA-Z]?[0-9]?)`)
var whiteMovesAlt = regexp.MustCompile(`([0-9]+\.[a-zA-Z-]+[0-9]?[a-zA-Z]?[0-9]?)`)
var blackMoves = regexp.MustCompile(`[^.] ([a-zA-Z][^ ]+) `)
var splitNotes = regexp.MustCompile(`,"([a-zA-Z0-9!\. \-\,\'’]+)"`)
var splitNotesNum = regexp.MustCompile(`([0-9]?[0-9]?[0-9]),"`)
var totalWritten = 0
var ctx context.Context
var cancel context.CancelFunc

func GetGame(url string) (string, string, error) {
	response, err := http.Get(url)

	if err != nil {
		return "", "", fmt.Errorf("Unable to make HTTP request", err)
	}

	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	pgn, exists := doc.Find("[pgn]").First().Attr("pgn")

	if !exists {
		log.Fatal("Cannot find pgn attribute")
	}

	notes, ex := doc.Find("[notes]").First().Attr("notes")

	if !ex {
		log.Fatal("Cannot find pgn attribute")
	}

	return html.UnescapeString(pgn), html.UnescapeString(notes), nil
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

func concatNotes(game string, notes string) string {
	/*
	   Takes original game string, cuts off half leaving only moves
	   Selects with Regex the white & black moves
	   If returns nothing, retries without a space (some games have spaces between nums, some don't)
	   Merges the two lists together placing it in, 'moveList'
	*/

	game = strings.Split(game, "\"]\n\n")[1]
	var white = whiteMoves.FindAllStringSubmatch(game, -1)
	var black = blackMoves.FindAllStringSubmatch(game, -1)

	if len(white) == 0 {
		white = whiteMovesAlt.FindAllStringSubmatch(game, -1)
	}

	// Merges black & white
	var moveList []string
	for i := 0; i < len(white); i++ {
		moveList = append(moveList, white[i][0])

		if len(black) > i {
			moveList = append(moveList, black[i][1])
		}
	}

	/*
	   Finds the text of the notes inside of the string
	   Finds the numbers that are beside the notes
	   Adds it to a list of notes in a pattern of 'index, note'
	*/

	nlNotes := splitNotes.FindAllStringSubmatch(notes, -1)
	nlNum := splitNotesNum.FindAllStringSubmatch(notes, -1)

	var notesList []string
	for i := 0; i < len(nlNum); i++ {
		notesList = append(notesList, nlNum[i][1])

		notesList = append(notesList, nlNotes[i][1])
	}

	/*
	   Loops through every move
	   Loops through every note (skipping by to to only get index)
	   If a note matches up with a move, add it
	*/

	var concatenatedString strings.Builder
	for i := 0; i < len(moveList); i++ {
		concatenatedString.WriteString(moveList[i] + " ")
		for n := 0; n < len(notesList); n += 2 {
			idx, _ := strconv.Atoi(notesList[n])
			if i == idx-2 || i == 0 {
				concatenatedString.WriteString("{" + notesList[n+1] + "} ")
				break
			}
		}
	}

	// Last part puts back in the score at the end (eg. 1-0)
	return concatenatedString.String() + game[len(game)-3:]
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
		game, notes, err := GetGame(g)

		if err != nil {
			log.Printf("Skipping game %d - Failed to download %s because %s", i+1, g, err.Error())
		} else {
			log.Printf("Writing game %d - %s", i+1, g)
			totalWritten++
			gameStr := concatNotes(game, notes)
			f.WriteString(gameStr)

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
		FetchAndWriteGames(games, *pgnPtr)

		log.Printf("Wrote %d games from %s to file %s\n", totalWritten, url, *pgnPtr)
	}
}
