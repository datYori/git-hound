package main

import (
	"flag"
	"fmt"
	"github.com/ezekg/git-hound/Godeps/_workspace/src/github.com/fatih/color"
	"github.com/ezekg/git-hound/Godeps/_workspace/src/sourcegraph.com/sourcegraph/go-diff/diff"
	"os"
	"sync"
)

func main() {
	var (
		noColor = flag.Bool("no-color", false, "Disable color output")
		config  = flag.String("config", ".githound.yml", "Hound config file")
		bin     = flag.String("bin", "git", "Executable binary to use for git command")
	)

	flag.Parse()

	if *noColor {
		color.NoColor = true
	}

	hound := &Hound{Config: *config}
	git := &Command{Bin: *bin}

	if ok, _ := hound.New(); ok {
		out, _ := git.Exec("diff", "-U0", "--staged")
		fileDiffs, err := diff.ParseMultiFileDiff([]byte(out))
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		errs := make(chan error)
		var wg sync.WaitGroup

		for _, fileDiff := range fileDiffs {
			fileName := fileDiff.NewName
			hunks := fileDiff.GetHunks()

			for _, hunk := range hunks {
				wg.Add(1)

				go func() {
					errs <- func() error {
						defer func() { wg.Done() }()
						return hound.Sniff(fileName, hunk)
					}()
				}()
			}
		}

		go func() {
			wg.Wait()

			for err := range errs {
				if err != nil {
					fmt.Print(err)
					os.Exit(1)
				}
			}
		}()
	}

	out, code := git.Exec(flag.Args()...)
	fmt.Print(out)
	os.Exit(code)
}
