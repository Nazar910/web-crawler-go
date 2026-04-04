package main

import "unicode"

type TokenType int

const (
	String TokenType = iota
	Colon
	Eof
)

type Token struct {
	tokenType TokenType
	value     string
}

type scanner struct {
	buf    []byte
	pos    int
	cursor byte
}

func newScanner(input []byte) scanner {
	var cursor byte
	if len(input) > 0 {
		cursor = input[0]
	}

	return scanner{input, 0, cursor}
}

func (s *scanner) advance() {
	s.pos++

	if s.pos < len(s.buf) {
		s.cursor = s.buf[s.pos]
	}
}

func (s *scanner) skipWs() {
	for s.pos < len(s.buf) && unicode.IsSpace(rune(s.cursor)) {
		s.advance()
	}
}

func (s *scanner) skipComments() {
	for s.pos < len(s.buf) && s.cursor == '#' {
		for s.pos < len(s.buf) && s.cursor != '\n' {
			s.advance()
		}
		if s.cursor == '\n' {
			s.advance()
		}
	}
}

func (s *scanner) string() string {
	start := s.pos
	for s.pos < len(s.buf) && (s.cursor != ':' && !unicode.IsSpace(rune(s.cursor))) {
		s.advance()
	}

	return string(s.buf[start:s.pos])
}

func (s *scanner) NextToken() (Token, error) {
	s.skipWs()
	s.skipComments()

	if s.pos >= len(s.buf) {
		return Token{Eof, ""}, nil
	}

	switch s.cursor {
	case ':':
		s.advance()
		return Token{Colon, ":"}, nil
	default:
		strVal := s.string()
		return Token{String, strVal}, nil
	}
}
