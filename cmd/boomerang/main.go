package main

import (
	"flag"
	"log"

	"github.com/fblazt/boomerang/internal/app"
)

func main() {
	preview := flag.String("preview", "", "Launch component catalog preview mode")
	flag.Parse()

	opts := app.DefaultOptions()
	opts.Preview = *preview

	application := app.NewApp(opts)
	if err := application.Run(); err != nil {
		log.Fatalf("failed to run Boomerang: %v", err)
	}
}
