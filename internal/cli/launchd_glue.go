package cli

import "github.com/laidbackgeek/envonce/internal/launchd"

// launchd_glue.go funnels the cli layer's launchd calls into one thin layer,
// so realLaunchdClient can delegate to it and the implementation can be swapped later.

func ldBootstrap(p string) error       { return launchd.New().Bootstrap(p) }
func ldBootout(l string) error         { return launchd.New().Bootout(l) }
func ldIsLoaded(l string) bool         { return launchd.New().IsLoaded(l) }
func ldPrint(l string) (string, error) { return launchd.New().Print(l) }
func ldLabelFor(n string) string       { return launchd.New().LabelFor(n) }
