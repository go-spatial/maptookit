package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/go-spatial/atlante/insetmap"
	"github.com/go-spatial/atlante/internal/env"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	entryFlag  = flag.String("entry", "", "which entry to use from the config file. If not set, will be the default entry, or the first entry.")
	mdgid      = flag.String("mdgid", "V795G25492", "mdgid to generate svg for.")
	configFile = flag.String("config", "config.toml", "config file")
	boundary   = flag.Bool("boundary", false, "generate a boundary map instead of an adjoining sheets map")
	Maps       map[string]*insetmap.Boundary
	DefaultMap string
	css        = flag.String("css", "", "the css file to embed without the ext")
)

type InsetEntry struct {
	Description string
	*insetmap.Inset
}
type ConfigEntry struct {
	DBConnStr env.String `toml:"database"`
	Desc      env.String `toml:"desc"`
	insetmap.Config
}

type Config struct {
	Addr         env.String             `toml:"address"`
	DefaultEntry env.String             `tomal:"default"`
	Entries      map[string]ConfigEntry `toml:"entry"`
}

func main() {
	flag.Parse()
	ctx := context.Background()
	var config Config
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse config(%v): %v\n", *configFile, err)
		os.Exit(1)
		return
	}
	DefaultMap = string(config.DefaultEntry)
	Maps = make(map[string]*insetmap.Boundary)

	for name, entry := range config.Entries {
		if DefaultMap == "" {
			// Use the first map as the default
			DefaultMap = name
		}

		conn, err := pgxpool.Connect(context.Background(), string(entry.DBConnStr))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%v] Unable to connect to database: %v\n", name, err)
			os.Exit(1)
		}

		defer conn.Close()
		imap, err := insetmap.NewBoundary(conn, entry.Config, "", nil, "")
		if err != nil {
			panic(err)
		}
		Maps[name] = imap

	}

	entry := DefaultMap
	if entryFlag != nil && *entryFlag != "" {
		entry = *entryFlag
	}

	log.Println("Using entry:", entry)

	var svg string
	if !*boundary {
		inset, err := Maps[entry].Inset.For(ctx, *mdgid, *css)
		if err != nil {
			panic(err)
		}
		svg, err = inset.AsSVG(false, "")
		if err != nil {
			panic(err)
		}
	} else {
		bmap, err := Maps[entry].For(ctx, *mdgid, *css)
		if err != nil {
			panic(err)
		}
		svg, err = bmap.AsSVG("")
		if err != nil {
			panic(err)
		}

	}
	fmt.Println(svg)

}
