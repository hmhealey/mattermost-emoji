package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type EmojiInput struct {
	Emoji string 
	Description string
	Aliases []string
	Tags []string
}

type EmojiOutput struct {
	Name string `json:"name"`
	Unicode string `json:"unicode,omitempty"`
	Filename string `json:"filename,omitempty"`
}

func main() {
	var input []EmojiInput
	var output []interface{}

	reader := bufio.NewReader(os.Stdin)
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&input); err != nil {
		panic("failed to decode input")
	}

	for _, emoji := range input {
		unicode := ""
		filename := ""

		if emoji.Emoji != "" {
			bytes := []byte(emoji.Emoji)

			var codepoint []string
			for i := 0; i < len(bytes); {
				r, size := utf8.DecodeRune(bytes[i:])
				i += size

				// ignore variation selectors
				if r >= 0xfe00 && r <= 0xfe0f {
					continue;
				}

				codepoint = append(codepoint, fmt.Sprintf("%04x", r))
			}

			unicode = strings.Join(codepoint, "-")
		}

		for i, alias := range emoji.Aliases {
			if i != 0 && unicode == "" {
				filename = emoji.Aliases[0]
			}

			output = append(output, []interface{}{
				alias,
				EmojiOutput{
					Name: alias,
					Unicode: unicode,
					Filename: filename,
				},
			})
		}
	}

	// print as a list of pairs [name, emoji] so that it can be unmarshaled as an ES6 map
	if bytes, err := json.Marshal(output); err != nil {
		panic("failed to output")
	} else {
		fmt.Printf("%v", string(bytes))
	}
}