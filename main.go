package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"ariga.io/entviz/internal/entviz"
)

var (
	devURL         string
	globalUniqueID bool
)

func init() {
	flag.StringVar(&devURL, "dev-url", "sqlite3://file?mode=memory&cache=shared&_fk=1", "dev database to be used to generate the schema")
	flag.BoolVar(&globalUniqueID, "global-unique-id", false, "enable the Global Unique ID feature")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of ariga.io/entviz\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\tgo run -mod=mod ariga.io/entviz <Ent schema path>\nFlags:\n")
		flag.PrintDefaults()
	}
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	flag.Parse()
	schemaPath := flag.Arg(0)
	if flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(1)
	}
	parsedDialect, atlasDriverName, err := entviz.ParseDevURL(devURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid dev-url: %v\n", err)
		os.Exit(1)
	}
	hcl, err := entviz.GenerateHCLFromEntSchema(ctx, entviz.GenerateOptions{
		SchemaPath:     schemaPath,
		Dialect:        parsedDialect,
		DevURL:         devURL,
		GlobalUniqueID: globalUniqueID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	link, err := entviz.ShareHCL(ctx, hcl, atlasDriverName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Here is a public link to your schema visualization\n\t%s\n", link)
}
