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
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
)

// Replaced with:
//   -ldflags="-X main.version=$(git describe --always --long --dirty)"
var version string = "undefined"
var release string = "undefined"
var shouldDebug bool = false
var log *logrus.Logger

func random_word(dictionary []string) string {
	var (
		len_dictionary int64
		n *big.Int
		err error
	)

	len_dictionary = int64(len(dictionary))

	n, err = rand.Int(rand.Reader, big.NewInt(len_dictionary))
	if err != nil {
		log.Fatal("Error during rand.Int: ", err)
		panic(err)
	}

	return dictionary[int(n.Int64())]
}

func main() {

	var logMain *logrus.Logger = &logrus.Logger{
		Out: os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Level: logrus.DebugLevel,
	}

	var ptrShouldDebug *string

	ptrShouldDebug = flag.String("shouldDebug", "false", "Should output debug output")

	flag.Parse()

	switch strings.ToLower(*ptrShouldDebug) {
	case "true":
		shouldDebug = true
	case "false":
		shouldDebug = false
	default:
		logMain.Fatal("Error: shouldDebug is not true/false (%s)\n", *ptrShouldDebug)
	}

	var out io.Writer

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

	// curl --remote-name --location https://raw.githubusercontent.com/dwyl/english-words/master/words_dictionary.json
	// sed -i -e 's,: 1,,' -e 's,{,[,' -e 's,},],' words_dictionary.json
	content, err := ioutil.ReadFile("./words_dictionary.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
 
	var dictionary []string
	err = json.Unmarshal(content, &dictionary)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
		panic(err)
	}

	log.Printf("len(dictionary) = %v\n", len(dictionary))

	var word = random_word(dictionary)

	fmt.Printf("%v\n", word)

	os.Exit(0)
}
