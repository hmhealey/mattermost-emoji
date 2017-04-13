// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// EmojiOne has a couple emoji that they consider aliases while we use different images for both
var IGNORE_FROM_EMOJI_ONE = map[string]bool{
	"e-mail":      true,
	"email":       true,
	"city_sunset": true,
	"city-sunset": true,
}

var CUSTOM_CATEGORIES = map[string]string{
	"e-mail":      "objects",
	"city_sunset": "travel",
}

type EmojiInput struct {
	Emoji       string
	Description string
	Aliases     []string
	Tags        []string
}

type EmojiOneEmoji struct {
	Unicode   string   `json:"unicode"`
	Shortname string   `json:"shortname"`
	Category  string   `json:"category"`
	Aliases   []string `json:"aliases"`
}

type EmojiOutput struct {
	Aliases  []string `json:"aliases"`
	Filename string   `json:"filename"`
}

func GetUnicodeForInput(emoji *EmojiInput) string {
	unicode := ""

	if emoji.Emoji != "" {
		bytes := []byte(emoji.Emoji)

		var codepoint []string
		for i := 0; i < len(bytes); {
			r, size := utf8.DecodeRune(bytes[i:])
			i += size

			// ignore variation selectors
			if r >= 0xfe00 && r <= 0xfe0f {
				continue
			}

			codepoint = append(codepoint, fmt.Sprintf("%04x", r))
		}

		unicode = strings.Join(codepoint, "-")
	}

	return unicode
}

func GetEmojiOneEmoji(emoji *EmojiInput, emojiOneEmojis map[string]*EmojiOneEmoji) *EmojiOneEmoji {
	for _, alias := range emoji.Aliases {
		if IGNORE_FROM_EMOJI_ONE[alias] {
			continue
		}

		for _, emojiOneEmoji := range emojiOneEmojis {
			if emojiOneEmoji.Shortname == ":"+alias+":" {
				return emojiOneEmoji
			}

			for _, emojiOneEmojiAlias := range emojiOneEmoji.Aliases {
				if emojiOneEmojiAlias == ":"+alias+":" {
					return emojiOneEmoji
				}
			}
		}
	}

	return nil
}

func main() {
	var emojisInput []*EmojiInput
	if emojiJson, err := os.Open("emoji.json"); err != nil {
		panic(fmt.Sprintf("failed to open emoji.json, err=%v\n", err))
	} else if err := json.NewDecoder(emojiJson).Decode(&emojisInput); err != nil {
		panic(fmt.Sprintf("failed to decode input, err=%v\n", err))
	}

	emojiOneEmojis := make(map[string]*EmojiOneEmoji)
	if emojiOneJson, err := os.Open("emoji-one.json"); err != nil {
		panic(fmt.Sprintf("failed to open emoji-one.json, err=%v\n", err))
	} else if err := json.NewDecoder(emojiOneJson).Decode(&emojiOneEmojis); err != nil {
		panic(fmt.Sprintf("failed to decode emoji-one.json, err=%v\n", err))
	}

	emojis := make([]*EmojiOutput, len(emojisInput))

	var aliases []string
	aliasMap := make(map[string]int)

	// Even though we don't really need it, keep an array of codepoints so we can iterate and output in a consistent order
	var unicodeCodepoints []string
	unicodes := make(map[string]int)

	var categoryNames []string
	emojisByCategory := make(map[string][]int)

	for i, emoji := range emojisInput {
		emojis[i] = &EmojiOutput{Aliases: emoji.Aliases}

		unicode := GetUnicodeForInput(emoji)
		if unicode != "" {
			if otherI, found := unicodes[unicode]; found && i != otherI {
				fmt.Fprintf(os.Stderr, "Duplicate emojis %v and %v for codepoint %v\n", emojis[i], emojis[otherI], unicode)
			}

			unicodes[unicode] = i
			unicodeCodepoints = append(unicodeCodepoints, unicode)

			// Emojis with a unicode equivalent use the unicode codepoint as their filename
			emojis[i].Filename = unicode
		}

		for _, alias := range emoji.Aliases {
			if otherI, found := aliasMap[alias]; found && i != otherI {
				fmt.Fprintf(os.Stderr, "Duplicate emojis %v and %v for %v\n", emojis[i], emojis[otherI], alias)
			}

			aliases = append(aliases, alias)
			aliasMap[alias] = i
		}

		category := "custom"
		if customCategory, found := CUSTOM_CATEGORIES[emoji.Aliases[0]]; found {
			category = customCategory
		}

		emojiOneEmoji := GetEmojiOneEmoji(emoji, emojiOneEmojis)
		if emojiOneEmoji != nil {
			// alias := emojiOneEmoji.Shortname[1:len(emojiOneEmoji.Shortname)-1]

			// if otherI, found := aliasMap[alias]; found && i != otherI {
			// 	fmt.Fprintf(os.Stderr, "Duplicate emojis %v and %v for %v\n", emojis[i], emojis[otherI], alias)
			// }

			// for _, aliasWithColons := range emojiOneEmoji.Aliases {
			// 	alias := aliasWithColons[1:len(aliasWithColons)-1]

			// 	if otherI, found := aliasMap[alias]; found && i != otherI {
			// 		fmt.Fprintf(os.Stderr, "Duplicate emojis %v and %v for %v\n", emojis[i], emojis[otherI], alias)
			// 	}

			// 	aliases = append(aliases, alias)
			// 	aliasMap[alias] = i
			// }

			if emojiOneEmoji.Category != "" {
				category = emojiOneEmoji.Category
			}
		}

		if category != "" {
			found := false

			for _, foundCategory := range categoryNames {
				if foundCategory == category {
					found = true
					break
				}
			}

			if !found {
				categoryNames = append(categoryNames, category)
			}

			emojisByCategory[category] = append(emojisByCategory[category], i)
		}
	}

	emojisJson, err := json.Marshal(emojis)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal emojis, err=%v\n", err))
	}

	sort.Sort(sort.StringSlice(aliases))
	emojiIndicesByAlias := make([][]interface{}, len(aliases))
	for i, alias := range aliases {
		emojiIndicesByAlias[i] = []interface{}{
			alias,
			aliasMap[alias],
		}
	}

	emojiIndicesByAliasJson, err := json.Marshal(emojiIndicesByAlias)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal emoji indices by alias, err=%v\n", err))
	}

	emojiIndicesByUnicode := make([][]interface{}, 0, len(unicodes))
	for _, codepoint := range unicodeCodepoints {
		emojiIndicesByUnicode = append(emojiIndicesByUnicode, []interface{}{
			codepoint,
			unicodes[codepoint],
		})
	}

	emojiIndicesByUnicodeJson, err := json.Marshal(emojiIndicesByUnicode)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal emoji indices by unicode codepoint, err=%v\n", err))
	}

	categoryNamesJson, err := json.Marshal(categoryNames)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal category names, err=%v\n", err))
	}

	emojiIndicesByCategory := make([][]interface{}, 0, len(emojisByCategory))
	for _, category := range categoryNames {
		emojiIndicesByCategory = append(emojiIndicesByCategory, []interface{}{
			category,
			emojisByCategory[category],
		})
	}

	emojiIndicesByCategoryJson, err := json.Marshal(emojiIndicesByCategory)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal emoji indices by category, err=%v\n", err))
	}

	fmt.Printf(
		`// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

// This file is automatically generated. Make changes to it at your own risk.

/* eslint-disable */

export const Emojis = %v;

export const EmojiIndicesByAlias = new Map(%v);

export const EmojiIndicesByUnicode = new Map(%v);

export const CategoryNames = %v;

export const EmojiIndicesByCategory = new Map(%v);

/* eslint-enable */
`,
		string(emojisJson),
		string(emojiIndicesByAliasJson),
		string(emojiIndicesByUnicodeJson),
		string(categoryNamesJson),
		string(emojiIndicesByCategoryJson),
	)
}
