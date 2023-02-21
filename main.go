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

	sourceMap := SourceMap{}
	enumsToReplace := map[string]any{}
	replaces := map[string][]ReplaceArea{}

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

		if _, ok := sourceMap[path]; ok {
			continue
		}

		matched++

		err = loadSources(path, sourceMap)
		if err != nil {
			log.Fatal(err)
		}
	}

	// find enums
	for _, path := range globResults {
		replaces[path] = append(replaces[path], findEnumsToReplace(sourceMap[path], enumsToReplace)...)
	}

	// find idents
	for _, path := range globResults {
		replaces[path] = append(replaces[path], findIdents(sourceMap[path], enumsToReplace)...)
	}

	for _, path := range globResults {
		fReplaces := replaces[path]

		if err = writeFile(path, fReplaces); err != nil {
			log.Fatal(err)
		}
	}

	if matched == 0 {
		log.Fatalf("input %q matched no files, see: -help", inputFiles)
	}

	log.Printf("matched %d\n", matched)
}
