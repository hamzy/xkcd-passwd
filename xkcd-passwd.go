// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Replaced with:
//   -ldflags="-X main.version=$(git describe --always --long --dirty)"
var version string = "undefined"
var release string = "undefined"
var shouldDebug bool = false
var log *logrus.Logger

type CaseType int
const (
	CaseNone	CaseType = iota		// case - all lowercase
	CaseAlternate				// CaSe - first character is upper case, second is lowercase, repeat
	CaseCapitalise				// CASE - first character is uppercase, rest are lowercase
	CaseInvert				// cASE - first character is lowercase, rest are uppercase
	CaseUpper				// CASE - all uppercase
	CaseRandom				// cASe - every character is randomly upper or lower
)
const CaseLower CaseType = CaseNone

type SeparatorType int
const (
	SeparatorNone		SeparatorType = iota
	SeparatorRandom
	SeparatorCharacter
)

type PaddingType int
const (
	PaddingNone		PaddingType = iota
	PaddingFixed
	PaddingAdaptive
)

// https://www.digitalocean.com/community/tutorials/how-to-use-json-in-go
type JSON_Defaults struct {
	NumWords		int		`json:"num_words"`
	WordLengthMin		int		`json:"word_length_min"`
	WordLengthMax		int		`json:"word_length_max"`
	CaseTransform		string		`json:"case_transform"`
	SeparatorCharacter	string		`json:"separator_character"`
	SeparatorAlphabet	[]string	`json:"separator_alphabet"`
	PaddingDigitsBefore	int		`json:"padding_digits_before"`
	PaddingDigitsAfter	int		`json:"padding_digits_after"`
	PaddingType		string		`json:"padding_type"`
	PaddingCharacter	string		`json:"padding_character"`
	SymbolAlphabet		[]string	`json:"symbol_alphabet"`
	PaddingCharactersBefore	int		`json:"padding_characters_before"`
	PaddingCharactersAfter	int		`json:"padding_characters_after"`
	PadToLength		int		`json:"pad_to_length"`
	RandomIncrement		string		`json:"random_increment"`
}

type Defaults struct {
	WordDictionary		[]string
	NumWords		int
	WordLengthMin		int
	WordLengthMax		int
	CaseTransform		CaseType
	SeparatorCharacter	SeparatorType
	SeparatorAlphabet	[]string
	PaddingDigitsBefore	int
	PaddingDigitsAfter	int
	PaddingType		PaddingType
	PaddingCharacter	string
	SymbolAlphabet		[]string
	PaddingCharactersBefore	int
	PaddingCharactersAfter	int
	PadToLength		int
	RandomIncrement		string
}

const jsonData1 = `{
 "num_words": 3,
 "word_length_min": 4,
 "word_length_max": 8,
 "case_transform": "CAPITALISE",
 "separator_character": "RANDOM",
 "separator_alphabet": [
  "!",
  "@",
  "$",
  "%",
  "^",
  "&",
  "*",
  "-",
  "_",
  "+",
  "=",
  ":",
  "|",
  "~",
  "?",
  "/",
  ".",
  ";"
 ],
 "padding_digits_before": 3,
 "padding_digits_after": 2,
 "padding_type": "FIXED",
 "padding_character": "RANDOM",
 "symbol_alphabet": [
  "!",
  "@",
  "$",
  "%",
  "^",
  "&",
  "*",
  "-",
  "_",
  "+",
  "=",
  ":",
  "|",
  "~",
  "?",
  "/",
  ".",
  ";"
 ],
 "padding_characters_before": 2,
 "padding_characters_after": 2,
 "random_increment": "AUTO"
}`

func read_defaults(jsonData string) (Defaults, error) {

	var json_defaults JSON_Defaults
	var defaults Defaults
	var err error
	err = json.Unmarshal([]byte(jsonData), &json_defaults)
	if err != nil {
		return Defaults{}, err
	}
	log.Printf("json_defaults: %+v\n", json_defaults)

	defaults.NumWords = json_defaults.NumWords
	defaults.WordLengthMin = json_defaults.WordLengthMin
	defaults.WordLengthMax = json_defaults.WordLengthMax
	json_defaults.CaseTransform = strings.ToLower(json_defaults.CaseTransform)
	switch json_defaults.CaseTransform {
	case "none":		defaults.CaseTransform = CaseNone
	case "alternate":	defaults.CaseTransform = CaseAlternate
	case "capitalise":	defaults.CaseTransform = CaseCapitalise
	case "invert":		defaults.CaseTransform = CaseInvert
	case "upper":		defaults.CaseTransform = CaseUpper
	case "lower":		defaults.CaseTransform = CaseLower
	case "random":		defaults.CaseTransform = CaseRandom
	default:
		return Defaults{}, errors.New(fmt.Sprintf("Error: Unknown CaseType: %v", json_defaults.CaseTransform))
	}
	defaults.SeparatorAlphabet = json_defaults.SeparatorAlphabet
	json_defaults.SeparatorCharacter = strings.ToLower(json_defaults.SeparatorCharacter)
	switch json_defaults.SeparatorCharacter {
	case "none":	defaults.SeparatorCharacter = SeparatorNone
	case "random":	defaults.SeparatorCharacter = SeparatorRandom
	default:
		if len(json_defaults.SeparatorCharacter) > 1 {
			return Defaults{}, errors.New(fmt.Sprintf("Error: Unknown SeparatorCharacter: %v", json_defaults.SeparatorCharacter))
		}
		defaults.SeparatorCharacter = SeparatorCharacter
		defaults.SeparatorAlphabet = make([]string, 1, 1)
		defaults.SeparatorAlphabet[0] = json_defaults.SeparatorCharacter
	}
	defaults.PaddingDigitsBefore = json_defaults.PaddingDigitsBefore
	defaults.PaddingDigitsAfter = json_defaults.PaddingDigitsAfter
	json_defaults.PaddingType = strings.ToLower(json_defaults.PaddingType)
	switch json_defaults.PaddingType {
	case "none":		defaults.PaddingType = PaddingNone
	case "fixed":		defaults.PaddingType = PaddingFixed
	case "adaptive":	defaults.PaddingType = PaddingAdaptive
	default:
		return Defaults{}, errors.New(fmt.Sprintf("Error: Unknown PaddingType: %v", json_defaults.PaddingType))
	}
	defaults.PaddingCharacter = json_defaults.PaddingCharacter
	defaults.SymbolAlphabet = json_defaults.SymbolAlphabet
	defaults.PaddingCharactersBefore = json_defaults.PaddingCharactersBefore
	defaults.PaddingCharactersAfter = json_defaults.PaddingCharactersAfter
	defaults.PadToLength = json_defaults.PadToLength
	defaults.RandomIncrement = json_defaults.RandomIncrement

	return defaults, err
}

func read_dictionary() ([]string, error) {

	var dictionary []string
	var err error

	// curl --remote-name --location https://raw.githubusercontent.com/dwyl/english-words/master/words_dictionary.json
	// sed -i -e 's,: 1,,' -e 's,{,[,' -e 's,},],' words_dictionary.json
	content, err := ioutil.ReadFile("./words_dictionary.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
 
	err = json.Unmarshal(content, &dictionary)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
		panic(err)
	}

	return dictionary, err

}

func random_separator(defaults Defaults) string {

	var (
		len_dictionary int64
		n *big.Int
		err error
	)

	len_dictionary = int64(len(defaults.SeparatorAlphabet))

	n, err = rand.Int(rand.Reader, big.NewInt(len_dictionary))
	if err != nil {
		log.Fatal("Error during rand.Int: ", err)
		panic(err)
	}

	return defaults.SeparatorAlphabet[int(n.Int64())]

}

func random_inner_word(defaults Defaults) string {

	var (
		len_dictionary int64
		n *big.Int
		err error
	)

	len_dictionary = int64(len(defaults.WordDictionary))

	n, err = rand.Int(rand.Reader, big.NewInt(len_dictionary))
	if err != nil {
		log.Fatal("Error during rand.Int: ", err)
		panic(err)
	}

	return defaults.WordDictionary[int(n.Int64())]

}

func random_word(defaults Defaults) string {

	var word string

	word = random_inner_word(defaults)
	for len(word) < defaults.WordLengthMin || len(word) > defaults.WordLengthMax {
		word = random_inner_word(defaults)
	}

	switch defaults.CaseTransform {
	case CaseLower:
		word = strings.ToLower(word)
	case CaseAlternate:
		chars := []rune{}
		for i, r := range word {
//			if len(chars) == 0 || !unicode.IsLetter(chars[len(chars) - 1]) || unicode.IsLower(chars[len(chars) - 1]) {
			if i % 2 == 0 {
				chars = append(chars, unicode.ToUpper(r))
			} else {
				chars = append(chars, unicode.ToLower(r))
			}
		}
		word = string(chars)
	case CaseCapitalise:
		chars := []rune{}
		for _, r := range word {
			chars = append(chars, unicode.ToUpper(r))
		}
		word = string(chars)
	case CaseInvert:
		chars := []rune{}
		for i, r := range word {
			if i == 0 {
				chars = append(chars, unicode.ToLower(r))
			} else {
				chars = append(chars, unicode.ToUpper(r))
			}
		}
		word = string(chars)
	case CaseUpper:
		word = strings.ToUpper(word)
	case CaseRandom:
		var (
			n *big.Int
			err error
		)
		chars := []rune{}
		for _, r := range word {
		word = string(chars)
			n, err = rand.Int(rand.Reader, big.NewInt(2))
			if err != nil {
				log.Fatal("Error during rand.Int: ", err)
				panic(err)
			}
			if n.Int64() == 0 {
				chars = append(chars, unicode.ToLower(r))
			} else {
				chars = append(chars, unicode.ToUpper(r))
			}
		}
		word = string(chars)
	}

	return word
}

func random_digit(num_digits int) string {

	var (
		m int64
		n *big.Int
		err error
	)

	m = int64(math.Pow10(num_digits))

	n, err = rand.Int(rand.Reader, big.NewInt(m))
	if err != nil {
		log.Fatal("Error during rand.Int: ", err)
		panic(err)
	}

	return strconv.Itoa(int(n.Int64()))

}

func generate_output(defaults Defaults) (error) {

	var (
//		padding string
		separator string
		err error
	)

//	padding = random_padding(defaults)
	separator = random_separator(defaults)

//	for i := 0; i < defaults.PaddingCharactersBefore; i++ {
//		fmt.Printf("%v", padding)
//	}

	if defaults.PaddingDigitsBefore > 0 {
		fmt.Printf("%v", random_digit(defaults.PaddingDigitsBefore))
		fmt.Printf("%v", separator)
	}

	for i := 0; i < defaults.NumWords; i++ {
		fmt.Printf("%v", random_word(defaults))
		if i < defaults.NumWords - 1 {
			fmt.Printf("%v", separator)
		}
	}

	if defaults.PaddingDigitsAfter > 0 {
		fmt.Printf("%v", separator)
		fmt.Printf("%v", random_digit(defaults.PaddingDigitsAfter))
	}

	fmt.Printf("\n")

	return err
}

func main() {

	var (
		logMain *logrus.Logger = &logrus.Logger{
			Out: os.Stderr,
			Formatter: new(logrus.TextFormatter),
			Level: logrus.DebugLevel,
		}
		ptrShouldDebug *string
		out io.Writer
		defaults Defaults
		dictionary []string
		err error
	)

	ptrShouldDebug = flag.String("shouldDebug", "false", "Should output debug output")

	flag.Parse()

	switch strings.ToLower(*ptrShouldDebug) {
	case "true":
		shouldDebug = true
	case "false":
		shouldDebug = false
	default:
		logMain.Fatal(fmt.Sprintf("Error: shouldDebug is not true/false (%s)\n", *ptrShouldDebug))
	}

	if shouldDebug {
		out = os.Stderr
	} else {
		out = io.Discard
	}
	log = &logrus.Logger{
		Out: out,
		Formatter: new(logrus.TextFormatter),
		Level: logrus.DebugLevel,
	}

	log.Printf("version = %v\nrelease = %v\n", version, release)

	defaults, err = read_defaults(jsonData1)
	if err != nil {
		log.Fatal("Error reading defaults: ", err)
	}

	dictionary, err = read_dictionary()
	if err != nil {
		log.Fatal("Error reading dictionary: ", err)
	}
	log.Printf("len(dictionary) = %v\n", len(dictionary))
	defaults.WordDictionary = dictionary

	err = generate_output(defaults)
	if err != nil {
		log.Fatal("Error generating output: ", err)
	}

	os.Exit(0)
}
