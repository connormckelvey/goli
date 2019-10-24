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
	fmt.Println(generate(ast))
	return []byte{}
}

type gen func(args ...interface{}) string

var generators map[string]gen

func init() {
	generators = map[string]gen{
		"package": func(args ...interface{}) string {
			if len(args) > 1 {
				panic("too many args")
			}
			return fmt.Sprintf("package %s", args[0].(string))
		},
		"import": func(args ...interface{}) string {
			template := `
	import (
		%s
	)`

			imports := []string{}
			for _, node := range args {
				n := node.(*Node)
				line := []string{}
				for _, c := range n.Children {
					line = append(line, c.(string))
				}
				imports = append(imports, strings.Join(line, " "))

			}
			return strings.TrimSpace(fmt.Sprintf(template, strings.Join(imports, "\n\t")))
		},
		"defn": func(args ...interface{}) string {

			nameReturn := args[0].(string)
			params := args[1]
			nameReturnParts := strings.Split(nameReturn, ":")
			name := nameReturnParts[0]
			returnType := ""
			if len(nameReturnParts) == 2 {
				returnType = nameReturnParts[1]
			}
			if returnType != "" {
				returnType = " " + returnType
			}
			paramsStr := parseDefnParams(params.(*Node).Children)

			output := fmt.Sprintf("\nfunc %s(%s)%s {", name, paramsStr, returnType)

			n := &Node{
				Children: args[2:],
			}
			output += generate(n)
			output += "\n}"

			return output
		},
		"default": func(args ...interface{}) string {
			call := args[0].(string)
			callParts := strings.Split(call, "/")
			goCall := strings.Join(callParts, ".")
			goCallArgs := []string{}

			for _, a := range args[1:] {
				goCallArgs = append(goCallArgs, a.(string))
			}
			goCallArgsStr := strings.Join(goCallArgs, ", ")
			output := "\n" + goCall + "(" + goCallArgsStr + ")"
			return output
		},
	}
}

func generate(tree *Node) string {
	children := tree.Children
	var i int
	var child interface{}

	output := ""

	for i < len(children) {
		child = children[i]
		if child == nil {
			fmt.Println("nil")
			i++
			continue
		}
		switch v := child.(type) {
		case string:
			args := children[1:]
			if generator, ok := generators[v]; ok {
				result := generator(args...)
				output = fmt.Sprintf("%s\n%s", output, result)
			} else {
				newargs := []interface{}{v}
				newargs = append(newargs, args...)
				result := generators["default"](newargs...)
				output = fmt.Sprintf("%s\n%s", output, result)
			}

			i += len(children)
			continue
		case *Node:
			output = fmt.Sprintf("%s\n%s", output, generate(v))
			i++
			continue
		}
	}
	return output
}

type Node struct {
	// Value    string
	Children []interface{}
	parent   *Node
}

func parseDefnParams(params []interface{}) string {
	paramsSlice := []string{}
	for _, param := range params {
		p := param.(string)
		pParts := strings.Split(p, ":")
		paramsSlice = append(paramsSlice, pParts[0]+" "+pParts[1])
	}
	return strings.Join(paramsSlice, ", ")
}

func buildAST(tokens []string) *Node {
	ast := &Node{}
	var parent *Node = nil
	cur := ast

	for _, token := range tokens {
		if strings.TrimSpace(token) == "" {
			continue
		}

		if token == "defn" {
			fmt.Printf("Token: %#v\n", token)
			fmt.Printf("%#v\n", cur)
			fmt.Printf("%#v\n", parent)
		}
		if token == "(" {
			node := &Node{
				parent: cur,
			}
			cur.Children = append(cur.Children, node)
			parent = cur
			cur = node
			continue
		} else if token == ")" {
			// once nested, and closing 2, ie )), parent points to the same place, probably need parent pointer fields on struct
			cur = cur.parent
			continue
		} else {
			cur.Children = append(cur.Children, token)
		}

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
