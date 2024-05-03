package main

import (
	"html"
	"strings"
	"testing"
)

var basicGame = "[PlyCount \"47\"]\n\n1. e4 e5 2.Nf3 Nc6 1-0"

func TestEmptyNotes(t *testing.T) {
	// empty
	res := ConcatNotes(basicGame, "[]")

	if res != basicGame {
		t.Fatalf("Empty notes should return the original pgn")
	}
}

func TestBasicNote(t *testing.T) {
	res := ConcatNotes(basicGame, "[0,\"nice game\"]")

	expected := "[PlyCount \"47\"]\n\n1. e4 {nice game} e5 2.Nf3 Nc6 1-0"
	if res != expected {
		t.Fatalf("Notes not properly included in game at move 0, got instead %s", res)
	}
}

func TestComplexGame1(t *testing.T) {
	game := html.EscapeString(`[Event "Vienna"]
[Site "Vienna AUH"]
[Date "1908.04.04"]
[EventDate "1908.03.23"]
[Round "10"]
[Result "1-0"]
[White "Akiba Rubinstein"]
[Black "Oldrich Duras"]
[ECO "D02"]
[WhiteElo "?"]
[BlackElo "?"]
[PlyCount "77"]

1.d4 d5 2.Nf3 c5 3.e3 Nf6 4.dxc5 Qa5+ 5.Nbd2 Qxc5 6.a3 Qc7 7.c4 dxc4 8.Nxc4 Nc6 9.b4 Bg4 10.Bb2 b5 11.Nce5 Nxe5 12.Nxe5 Bxd1 13.Bxb5+ Nd7 14.Bxd7+ Qxd7 15.Nxd7 Bh5 16.Ne5 Rc8 17.g4 Bg6 18.Nxg6 hxg6 19.Bd4 a6 20.Kd2 f6 21.Rac1 Rxc1 22.Rxc1 e5 23.Bc5 Rxh2 24.Bxf8 Kxf8 25.Ke2 e4 26.Rc6 Rg2 27.Rxa6 Rxg4 28.Ra7 Rg1 29.b5 Rb1 30.a4 g5 31.Rb7 Ra1 32.b6 Rxa4 33.Ra7 Rb4 34.b7 g4 35.Ra8+ Kf7 36.b8=Q Rxb8 37.Rxb8 Ke6 38.Re8+ Kf5 39.Kf1 1-0`)

	notes := `[0,"Notes by Carl Schlechter from \"Deutsche Schachzeitung\" 1908.",7,"? Bad, because this helps the opponent to develop. The right move is 4...e6, and if 5.b4? then 5...a5 6.c3 axb4 7.cxb4 b6 regaining the pawn.",13,"This also helps White’s development. Better was 7...e6.",19,"? This will be refuted by a nice combination by White, but Black already stands worse. If, for example, 10...e6, then 11.Rc1!, threatening b5.",20,"!",22,"!!",25,"Best. If 13...Kd8 14.Rxd1+ Kc8 15.Ba6+ Kb8 16.Nc6+ Qxc6 17.Be5+ Qd6 (17...Qc7 18.Rd8+ mate) 18.Rc1!! and mate next move.",26,"The simplest. White forces an endgame with a pawn plus. Stronger was 14.Rxd1 Rd8 15.Nxd7 Rxd7 (or 15...e6 16.Ne5+ Ke7 17.Nc6+, etc.) 16.Bxd7+ Kd8 17.Bb5+ Kc8 18.Ba6+ Kb8 19.Rc1!, followed by Be5, and wins.",43,"Or 22...Rxh2 23.Rc8+ Kf7 24.Ke2! e5 25.Bc5 Bxc5 26.Rxc5 followed by Ra5 winning the a-pawn.",48,"!",51,"If 26...a5 27.b5 followed by Ra6."]`

	res := ConcatNotes(game, notes)

	if !strings.Contains(res, "Notes by") {
		t.Fatalf("Unable to find initial comment in final game text")
	}

	if !strings.Contains(res, "followed by Ra6") {
		t.Fatalf("Unable to find last comment in final game text")
	}
}
