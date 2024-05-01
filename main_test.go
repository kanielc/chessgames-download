package main

import (
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
