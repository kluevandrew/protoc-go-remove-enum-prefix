package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var inputFiles string
	flag.StringVar(&inputFiles, "input", "", "pattern to match input file(s)")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")

	flag.Parse()

	if inputFiles == "" {
		log.Fatal("input file is mandatory, see: -help")
	}

	// Note: glob doesn't handle ** (treats as just one *). This will return
	// files and folders, so we'll have to filter them out.
	globResults, err := filepath.Glob(inputFiles)
	if err != nil {
		log.Fatal(err)
	}

	var matched int
	for _, path := range globResults {
		finfo, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if finfo.IsDir() {
			continue
		}

		// It should end with ".go" at a minimum.
		if !strings.HasSuffix(strings.ToLower(finfo.Name()), ".go") {
			continue
		}

		matched++

		indents, comments, err := parseFile(path, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err = writeFile(path, indents, comments); err != nil {
			log.Fatal(err)
		}
	}

	if matched == 0 {
		log.Fatalf("input %q matched no files, see: -help", inputFiles)
	}

	log.Printf("matched %d\n", matched)
}
