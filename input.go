package main

import (
	"errors"
	"os"
	"strconv"
	"unicode"
)

const BashLineVar = "COMP_LINE"
const BashCursorVar = "COMP_POINT"
const BashMaxLineSize = 4096

type BashInput struct {
	CursorPosition int
	CmdLine        string
	CmdName        *string
	CurrentWord    *string
	PreviousWord   *string
}

type BashParseState uint8

const (
	NADA BashParseState = iota
	InWord
	InQuote
	InDblQuote
)

func CreateCompletionInput() (*BashInput, error) {
	cmdLine := os.Getenv(BashLineVar)
	if len(cmdLine) == 0 {
		return nil, errors.New("missing BASH env variable: " + BashLineVar)
	}
	cursorPosStr := os.Getenv(BashCursorVar)
	if len(cursorPosStr) == 0 {
		return nil, errors.New("missing BASH env variable: " + BashCursorVar)
	}
	cursorPos, err := strconv.Atoi(cursorPosStr)
	if err != nil {
		return nil, err
	}
	commandName := getCommandNameFromInput(cmdLine)
	currentWord := getCurrentWord(cmdLine, cursorPos)
	previousWord := getPreviousWord(cmdLine, cursorPos)
	input := BashInput{cursorPos, cmdLine, commandName, currentWord, previousWord}
	return &input, nil
}

func getCommandNameFromInput(cmdLine string) *string {
	list := BashInputToList(cmdLine, BashMaxLineSize)
	if len(list) > 0 {
		return &list[0]
	} else {
		return nil
	}
}

func getCurrentWord(cmdLine string, cursorPosition int) *string {
	list := BashInputToList(cmdLine, cursorPosition)
	if len(list) >= 1 {
		return &list[len(list)-1]
	} else {
		return nil
	}
}

func getPreviousWord(cmdLine string, cursorPosition int) *string {
	list := BashInputToList(cmdLine, cursorPosition)
	if len(list) >= 2 {
		return &list[len(list)-2]
	} else {
		return nil
	}
}

func BashInputToList(cmdLine string, maxLen int) []string {
	var list []string

	var state = NADA
	var startOfWord = 0
	var i = 0
	var c rune = 0
	for i, c = range cmdLine {
		var gotWord = false
		switch state {
		case NADA:
			if !unicode.IsSpace(c) {
				// transition to new state
				switch c {
				case '"':
					state = InDblQuote
					startOfWord = i + 1
				case '\'':
					state = InQuote
					startOfWord = i + 1
				default:
					state = InWord
					startOfWord = i
				}
			}
		case InWord:
			// word ends with whitespace or equals (=)
			if unicode.IsSpace(c) || (c == '=') {
				gotWord = true
			}
		case InQuote:
			// keep going until quote
			gotWord = c == '\''
		case InDblQuote:
			// keep going until double-quote
			gotWord = c == '"'
		}
		if gotWord {
			word := cmdLine[startOfWord:i]
			list = append(list, word)
			// change state
			state = NADA
			gotWord = false
			startOfWord = 0
		}
		// make sure we only consider maxLen characters
		if i >= maxLen {
			break
		}
	}

	// check if we have a remaining word in the buffer
	if (state != NADA) && (startOfWord != 0) {
		// make sure we don't consider more characters than allowed
		word := cmdLine[startOfWord : i+1]
		list = append(list, word)
	}

	return list
}
