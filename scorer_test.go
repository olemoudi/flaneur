package main

import (
	"testing"
)

func TestNewBloomScorer(t *testing.T) {

	bs := NewBloomScorer(4)

	if len(bs.filters) != 4 {
		t.Fatalf("Unexpected Scorer filters size. Was", len(bs.filters), "Expected 4")
	}

	input1 := []string{"a", "b"}

	if bs.Score(input1) != 1 {
		t.Fatalf("Score after first test is not 1")
	}

	if bs.Score(input1) != 0 {
		t.Fatalf("Score after second test is not 0")
	}

	input2 := []string{"a", "c"}

	if bs.Score(input2) != 0.5 {
		t.Fatalf("Score changing only half variables is not 0.5")
	}

	input3 := []string{}

	if bs.Score(input3) == 1 {
		t.Fatalf("Empty element should always present")
	}

}
