package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

// WordCount This comment is only here to satisfy Visual Studio Code
/*
	WordCount serves as a tuple for a word and how many times it's counted.
	Ie the WordCount WordCount{"donkey", 12} means "donkey" appeared 12 times
*/
type WordCount struct {
	Word  string
	Count int
}

/*
	Purpose
	Takes a list of all of words, iterates through them, and then stores that
	information in a map. This function is used to count all of the words in
	file containing Shakespeare's works.

	Parameters
	The list of words to count

	Returns
	A map of the form x -> y where the word x appeared y times in the given
	list of words
*/
func countWords(words []string) map[string]int {
	frequencyMap := make(map[string]int)

	for _, word := range words {
		count, exists := frequencyMap[word]

		// We've seen the word and increase the count
		if exists {
			frequencyMap[word] = count + 1
			// The word's new and we have to start counting
		} else {
			frequencyMap[word] = 1
		}

	}

	return frequencyMap
}

/*
	Purpose
	Takes a file path, reads the contents of the, gets rid of non alphabetic
	characters, normalizes the text by decapitalizing all of its characters,
	and then parses all of the words. The parsing is done using regex. This
	function is used to parse the file of Shakespeare's works.

	Parameters
	The path to a file relative to the working directory

	Returns
	A list of the words in the file given
*/
func parseFile(path string) []string {
	// Try to read the file
	bytes, readError := ioutil.ReadFile(path)
	if readError != nil {
		panic(readError)
	}
	fileAsString := string(bytes)
	log.Printf("Successfully read file")

	// Filter out non alphabetic characters
	nonAlphabeticRegex := regexp.MustCompile("[^a-zA-Z\\s]")
	fileAlpha := nonAlphabeticRegex.ReplaceAllString(fileAsString, "")
	log.Printf("Successfully filtered file")

	// Make everything lowercase
	fileAlpha = strings.ToLower(fileAlpha)

	// Parse into words
	whitespaceRegex := regexp.MustCompile("\\s+")
	fileAsWords := whitespaceRegex.Split(fileAlpha, -1)
	log.Printf("Successfully parsered file")

	return fileAsWords
}

/*
	Purpose
	Takes a prefix and completes the word with the most likely words. The
	completion is done by finding the most frequently used words in the given
	frequency map, and then putting them into a list where the most frequently
	used words come first

	Parameters
	prfix - the beginning of the word. Ie "th"
	frequencies - a map of the form x -> where the word x occurs y times in the
				  training data used to complete the word

	Returns
	A list of strings. These strings all begin with the given prefix. The
	strings appear in the how frequently they were used. Ie the first string
	was used most often and the last string was used least often.
*/
func complete(prefix string, frequencies map[string]int) []string {

	// Find the words that match and how many times they occur
	wordCounts := make([]WordCount, 0)

	for word, count := range frequencies {
		if strings.HasPrefix(word, prefix) {
			wordCounts = append(wordCounts, WordCount{Word: word, Count: count})
		}
	}

	// Sort the matches in decreasing order
	sort.SliceStable(wordCounts, func(i, j int) bool {
		return !(wordCounts[i].Count < wordCounts[j].Count)
	})
	log.Println("# of matches: ", len(wordCounts))

	// Convert the matches into a list of strings
	matches := make([]string, len(wordCounts))
	for i := 0; i < len(wordCounts); i++ {
		matches[i] = wordCounts[i].Word
	}

	return matches
}

/*
	Purpose
	Takes the first n strings of a list. If n is larger than the amount of
	elements of the list, then the original list is returned.

	Parameters
	The list to take the first n of. Assuming this is positive

	Returns
	The first n elements in the list or the entire list if n > len(n)

*/
func firstN(words []string, n int) []string {
	// Filter to the top n hits if needed
	if n < len(words) {
		return words[:n]
	}

	return words
}

/*
	Purpose
	Handles the api part of the program. Given the expected request, the top 25
	best autcompletions will be returned given an error doesn't happen. If the
	request isn't as expected, a message will be returned

	Thanks: https://tutorialedge.net/golang/creating-restful-api-with-golang/

	Parameters
		w - the response write
		r - the request
*/
func respond(w http.ResponseWriter, r *http.Request) {
	log.Println("Entering response handler...")

	// Get the url
	url, err := url.Parse(r.RequestURI)
	if err != nil {
		panic(err)
	}

	// Validate the request
	term := r.URL.Query().Get("term")
	supportedMethod := r.Method == "GET"
	validEndpoint := url.Path == "/autocomplete"
	includedTerm := term != ""
	validRequest := supportedMethod && validEndpoint && includedTerm

	// Respond
	w.Header().Set("Content-Type", "application/json")
	if validRequest {
		// Find the matches, take the 1st 25, and derialize
		matches := complete(term, countMap)
		top25 := firstN(matches, 25)
		top25json, error := json.Marshal(top25)

		log.Printf("Prefix: %s Matches: %v", term, top25)

		// Try to respond
		if error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "Internal service error"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"matches":` + string(top25json) + `}`))
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Unsupported request"}`))

	}
}

/*
	Purpose
	Initializes the frequency map used to complete words. If the process of
	initialization, thie function must also read and parse the input file
	given via the command line
*/
func initCounts() map[string]int {
	// Access the input file via command line
	if len(os.Args) < 2 {
		log.Println("Please include a file path as the 1st cmd line argument")
		os.Exit(1)
	}

	// Read the file path
	filePath := os.Args[1]
	log.Println("Data file path: ", filePath)

	// Parse the file
	fileAsWords := parseFile(filePath)

	return countWords(fileAsWords)
}

// Each word and it's count in the file passed on startup
var countMap = initCounts()

func main() {
	log.Println("Starting server...")
	http.HandleFunc("/", respond)
	log.Fatal(http.ListenAndServe(":9000", nil))
	log.Println("Stopping server...")
}
