package main

import (
	"flag"
	"fmt"

	projectbuild "github.com/LikiPiki/tc002-apps/apps/tools/internal/build"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func runToolList(args []string) error {
	flags := flag.NewFlagSet("clockctl tool-list", flag.ContinueOnError)
	base, personal := config.Paths(flags)

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected tool-list arguments")
	}

	c, err := config.Load(*base, *personal)
	if err != nil {
		return err
	}
	targets, err := projectbuild.Select("apps/tools", c.Tools.Apps)
	if err != nil {
		return err
	}

	for _, target := range targets {
		fmt.Println(target)
	}

	return nil
}
