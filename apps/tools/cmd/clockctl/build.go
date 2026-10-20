package main

import (
	"flag"
	"fmt"

	projectbuild "github.com/LikiPiki/tc002-apps/apps/tools/internal/build"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func runBuildList(command string, args []string) error {
	flags := flag.NewFlagSet("clockctl "+command, flag.ContinueOnError)
	base, personal := config.Paths(flags)
	group := flags.String("group", "all", "all, server, or tools")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected build-list arguments")
	}
	c, err := config.Load(*base, *personal)
	if err != nil {
		return err
	}
	if command == "server-list" {
		*group = "server"
	}
	targets, err := projectbuild.Select("apps", c.Server.Apps, c.Tools.Apps, *group)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if command == "server-list" {
			fmt.Println(target.Name)
		} else {
			fmt.Println(target.String())
		}
	}
	return nil
}
