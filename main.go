package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

func main() {
	program, err := ioutil.ReadFile("main.goli")
	if err != nil {
		log.Fatal(err)
	}
	result := parse(program)
	fmt.Println(string(result))
}

func parse(program []byte) []byte {
	program = prepare(program)
	tokens := tokenize(program)
	ast := buildAST(tokens)
	j, _ := json.MarshalIndent(ast, "", "\t")
	fmt.Println(string(j))
	return []byte{}
}

func buildAST(tokens []string) interface{} {
	ast := []interface{}{}
	parent := &ast
	cur := &ast

	for _, token := range tokens {
		if strings.TrimSpace(token) == "" {
			continue
		}

		if token == "(" {
			node := []interface{}{}
			*cur = append(*cur, &node)
			parent = cur
			cur = &node
			continue
		} else if token == ")" {
			cur = parent
			continue
		}
		*cur = append(*cur, token)

	}
	return ast
}

func tokenize(program []byte) []string {
	input := string(program)
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")
	input = strings.TrimSpace(input)
	tokens := strings.Split(input, " ")

	return tokens
}

func prepare(input []byte) []byte {
	quoteMap, input, err := preserveQuotes(input)
	if err != nil {
		log.Println(err)
	}
	input, err = stripComments(input)
	if err != nil {
		log.Println(err)
	}
	input, err = restoreQuotes(input, quoteMap)
	return input
}

func preserveQuotes(input []byte) (map[string]string, []byte, error) {
	quoteMap := map[string]string{}

	patternb64 := "IlteIlxcXSooXFwoLnxcbilbXiJcXF0qKSoifCdbXidcXF0qKFxcKC58XG4pW14nXFxdKikqJ3xgW15gXFxdKihcXCgufFxuKVteYFxcXSopKmA="
	pattern, err := base64.StdEncoding.DecodeString(patternb64)
	if err != nil {
		return nil, input, err
	}

	r, err := regexp.Compile(string(pattern))
	if err != nil {
		return nil, input, err
	}

	output := r.ReplaceAllStringFunc(string(input), func(s string) string {
		u := uuid.New().String()
		quoteMap[u] = s
		return u

	})

	return quoteMap, []byte(output), nil
}

func stripComments(input []byte) ([]byte, error) {
	var output []byte
	bufreader := bufio.NewReader(bytes.NewReader(input))
	for {
		line, err := bufreader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				output = append(output, line...)
				break
			}
			return output, err
		}
		linereader := bufio.NewReader(bytes.NewReader(line))
		toComment, err := linereader.ReadBytes(';')
		if err != nil {
			if err == io.EOF {
				// No comment on line
				output = append(output, toComment...)
				continue
			}
		}
		output = append(output, toComment[:len(toComment)-1]...)
		output = append(output, '\n')
	}
	return output, nil
}

func restoreQuotes(input []byte, quoteMap map[string]string) ([]byte, error) {
	for k, v := range quoteMap {
		r, err := regexp.Compile(k)
		if err != nil {
			return nil, err
		}
		input = r.ReplaceAll(input, []byte(v))
	}
	return input, nil
}
