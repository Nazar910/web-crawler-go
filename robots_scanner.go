package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

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
	r *bufio.Reader
}

func newScanner(reader io.Reader) (scanner, error) {
	r := bufio.NewReader(reader)

	_, err := r.Peek(1)

	if err == io.EOF {
		return scanner{}, fmt.Errorf("empty robots.txt input: %w", err)
	}

	return scanner{r: r}, nil
}

func (s *scanner) skipWs() error {
	for {
		b, err := s.r.Peek(1)

		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		ch := b[0]

		if unicode.IsSpace(rune(ch)) {
			s.r.ReadByte()
		} else if ch == '#' {
			err := s.skipComments()

			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}
}

func (s *scanner) skipComments() error {
	for {
		b, err := s.r.Peek(1)

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		ch := b[0]
		s.r.ReadByte()
		if ch == '\n' {
			return nil
		}
	}
}

func (s *scanner) string() (string, error) {
	var sb strings.Builder
	for {
		b, err := s.r.Peek(1)

		if err != nil {
			if err == io.EOF {
				break
			}

			return sb.String(), err
		}

		if b[0] == ':' || unicode.IsSpace(rune(b[0])) {
			break
		}

		sb.WriteByte(b[0])
		s.r.ReadByte()
	}

	return sb.String(), nil
}

func (s *scanner) NextToken() (Token, error) {
	err := s.skipWs()

	if err != nil {
		return Token{}, err
	}

	b, err := s.r.Peek(1)

	if err != nil {
		if err == io.EOF {
			return Token{Eof, ""}, nil
		}
		return Token{}, err
	}

	switch b[0] {
	case ':':
		s.r.ReadByte()
		return Token{Colon, ":"}, nil
	default:
		strVal, err := s.string()
		return Token{String, strVal}, err
	}
}
