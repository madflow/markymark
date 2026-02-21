package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/madflow/markymark/internal/finder"
	"github.com/madflow/markymark/internal/markdown"
	"github.com/madflow/markymark/internal/server"
)

func main() {
	var watchMode bool
	flag.BoolVar(&watchMode, "watch", false, "Watch the markdown file and reload the browser on changes")
	flag.BoolVar(&watchMode, "w", false, "Watch the markdown file and reload the browser on changes (shorthand)")
	flag.Parse()

	filePath := resolveFilePath()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatal(err)
	}
	baseDir := filepath.Dir(absPath)

	// componentFn re-reads and re-renders the file on every call.
	// In normal mode it is called once; in watch mode it is called on each
	// file-change event so the server always serves fresh content.
	componentFn := func() templ.Component {
		content, err := os.ReadFile(absPath)
		if err != nil {
			log.Printf("read %s: %v", absPath, err)
			return TemplatePage("Markdown", "", watchMode)
		}
		doc := markdown.Parse(content)
		body := markdown.Render(doc)
		return TemplatePage("Markdown", string(body), watchMode)
	}

	// Derive the allowedImages set from the initial parse.
	initialContent, err := os.ReadFile(absPath)
	if err != nil {
		log.Fatal(err)
	}
	doc := markdown.Parse(initialContent)
	allowedImages := markdown.ExtractRelativeImages(doc)

	server.New(componentFn, baseDir, allowedImages, watchMode, absPath).Start("localhost:3000")
}

func resolveFilePath() string {
	args := flag.Args()
	if len(args) > 0 {
		return args[0]
	}
	path := finder.FindReadme()
	if path == "" {
		log.Fatal("No README.md file found")
	}
	return path
}
