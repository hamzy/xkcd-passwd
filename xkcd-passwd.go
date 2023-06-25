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

// Based on https://xkpasswd.net/

// go vet && go build -ldflags="-X main.version=$(git describe --always --long --dirty) -X main.release=$(git tag --sort=-version:refname | head -n1)" .

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
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// Replaced with:
//   -ldflags="-X main.version=$(git describe --always --long --dirty) -X main.release=$(git tag --sort=-version:refname | head -n1)"
var version string = "undefined"
var release string = "undefined"
var shouldDebug bool = false
var log *logrus.Logger

type CaseType int
const (
	CaseNone	CaseType = iota		// case - all lowercase
	CaseAlternate				// CaSe - first character is upper case, second is lowercase, repeat
	CaseCapitalise				// Case - first character is uppercase, rest are lowercase
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

type PaddingCharacter int
const (
	PaddingRandom		PaddingCharacter = iota		// Use symbol_alphabet
	PaddingSeparator					// Use SeparatorRandom result
	PaddingSpecified					// Use the string value
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
	PaddingCharacter	PaddingCharacter
	SymbolAlphabet		[]string
	PaddingCharactersBefore	int
	PaddingCharactersAfter	int
	PadToLength		int
}

func read_defaults(jsonData []byte) (Defaults, error) {

	var json_defaults JSON_Defaults
	var defaults Defaults
	var err error
	err = json.Unmarshal(jsonData, &json_defaults)
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
	defaults.SymbolAlphabet = json_defaults.SymbolAlphabet
	json_defaults.PaddingCharacter = strings.ToLower(json_defaults.PaddingCharacter)
	switch json_defaults.PaddingCharacter {
	case "random":		defaults.PaddingCharacter = PaddingRandom
	case "separator":	defaults.PaddingCharacter = PaddingSeparator
	default:
		if len(json_defaults.PaddingCharacter) > 1 {
			return Defaults{}, errors.New(fmt.Sprintf("Error: Unknown PaddingCharacter: %v", json_defaults.PaddingCharacter))
		}
		defaults.PaddingCharacter = PaddingSpecified
		defaults.SymbolAlphabet = make([]string, 1, 1)
		defaults.SymbolAlphabet[0] = json_defaults.PaddingCharacter
	}
	defaults.PaddingCharactersBefore = json_defaults.PaddingCharactersBefore
	defaults.PaddingCharactersAfter = json_defaults.PaddingCharactersAfter
	defaults.PadToLength = json_defaults.PadToLength

	return defaults, err
}

func read_dictionary(filename string) ([]string, error) {

	var dictionary []string
	var err error

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
		panic(err)
	}
 
	err = json.Unmarshal(content, &dictionary)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
		panic(err)
	}

	return dictionary, err

}

func random_padding(defaults Defaults) string {

	var (
		len_dictionary int64
		n *big.Int
		err error
	)

	len_dictionary = int64(len(defaults.SymbolAlphabet))

	n, err = rand.Int(rand.Reader, big.NewInt(len_dictionary))
	if err != nil {
		log.Fatal("Error during rand.Int: ", err)
		panic(err)
	}

	return defaults.SymbolAlphabet[int(n.Int64())]

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
		for i, r := range word {
			if i == 0 {
				chars = append(chars, unicode.ToUpper(r))
			} else {
				chars = append(chars, unicode.ToLower(r))
			}
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

func random_digits(num_digits int) string {

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

	return fmt.Sprintf("%0*d", num_digits, n.Int64())

}

func generate_output(defaults Defaults) (error) {

	var (
		builder strings.Builder
		result string
		separator string
		padding string
		err error
	)

	separator = random_separator(defaults)

	if defaults.PaddingType == PaddingFixed || defaults.PaddingType == PaddingAdaptive {
		if defaults.PaddingCharacter == PaddingRandom {
			padding = random_padding(defaults)
		} else if defaults.PaddingCharacter == PaddingSeparator {
			padding = separator
		} else if defaults.PaddingCharacter == PaddingSpecified {
			padding = defaults.SymbolAlphabet[0]
		}
	}

	if defaults.PaddingType == PaddingFixed {
		for i := 0; i < defaults.PaddingCharactersBefore; i++ {
			fmt.Fprintf(&builder, "%v", padding)
		}
	}

	if defaults.PaddingDigitsBefore > 0 {
		fmt.Fprintf(&builder, "%v", random_digits(defaults.PaddingDigitsBefore))
		fmt.Fprintf(&builder, "%v", separator)
	}

	for i := 0; i < defaults.NumWords; i++ {
		fmt.Fprintf(&builder, "%v", random_word(defaults))
		if i < defaults.NumWords - 1 {
			fmt.Fprintf(&builder, "%v", separator)
		}
	}

	if defaults.PaddingDigitsAfter > 0 {
		fmt.Fprintf(&builder, "%v", separator)
		fmt.Fprintf(&builder, "%v", random_digits(defaults.PaddingDigitsAfter))
	}

	if defaults.PaddingType == PaddingFixed {
		for i := 0; i < defaults.PaddingCharactersAfter; i++ {
			fmt.Fprintf(&builder, "%v", padding)
		}
	}

	result = builder.String()
	log.Printf("len builder = %v", len(result))

	if defaults.PaddingType == PaddingAdaptive {
		log.Printf("PadToLength = %v", defaults.PadToLength)
		if len(result) > defaults.PadToLength {
			result = result[1:defaults.PadToLength+1]
		} else if len(result) < defaults.PadToLength {
			var length = defaults.PadToLength - len(result)
			for i := 0; i < length; i++ {
				fmt.Fprintf(&builder, "%v", padding)
			}
			result = builder.String()
		}
	}

	fmt.Printf("%v\n", result)

	return err
}

func main() {

	var (
		logMain *logrus.Logger = &logrus.Logger{
			Out: os.Stderr,
			Formatter: new(logrus.TextFormatter),
			Level: logrus.DebugLevel,
		}
		ptrShouldVerson *bool
		ptrShouldDebug *string
		num_passwords = 1
		args []string
		out io.Writer
		homeDir string
		filename string
		defaultFilename string
		jsonData []byte
		defaults Defaults
		err error
	)

	ptrShouldVerson = flag.Bool("version", false, "Should output program version")
	ptrShouldDebug = flag.String("shouldDebug", "false", "Should output debug output")

	flag.Parse()
	args = flag.Args()

	if *ptrShouldVerson {
		fmt.Println("version =", version)
		fmt.Println("release =", release)
		os.Exit(0)
	}

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

	log.Printf("flag.Args = %v\n", args)
	if len(args) == 1 {
		num_passwords, err = strconv.Atoi(args[0])
		if err != nil {
			logMain.Fatal("Error during strconv.Atoi: ", err)
		}
	} else if len(args) != 0 {
		logMain.Fatal(fmt.Sprintf("Error: Only one argument is allowed\n"))
	}

	log.Printf("version = %v\nrelease = %v\n", version, release)

	// Find home
	homeDir, err = os.UserHomeDir()
	if err != nil {
		logMain.Fatal("Error when callin os.UserHomeDir: ", err)
		panic(err)
	}
	log.Printf("homeDir = %v", homeDir)

	// Does our default file live at home?
	filename = filepath.Join(homeDir, ".xkcd-defaults.json")
	_, err = os.Stat(filename)
	log.Printf("os.Stat(\"%v\") = %v\n", filename, err)
	if err == nil {
		defaultFilename = filename
	} else {
		// Or does it live in the current directory?
		filename = ".xkcd-defaults.json"
		_, err = os.Stat(filename)
		log.Printf("os.Stat(\"%v\") = %v\n", filename, err)
		if err == nil {
			defaultFilename = filename
		}
	}

	// Read the .xkcd-defaults.json file
	jsonData, err = ioutil.ReadFile(defaultFilename)
	if err != nil {
		logMain.Fatal("Error when opening .xkcd-defaults.json: ", err)
		panic(err)
	}

	// Return the default struct from the file data
	defaults, err = read_defaults(jsonData)
	if err != nil {
		logMain.Fatal("Error reading defaults: ", err)
	}
	log.Printf("defaults: %+v\n", defaults)

	log.Printf("len(dictionary) = %v\n", len(dictionary))
	defaults.WordDictionary = dictionary

	for i := 0; i < num_passwords; i++ {
		// Generate the password based on the data in the defaults structure
		err = generate_output(defaults)
		if err != nil {
			logMain.Fatal("Error generating output: ", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}
