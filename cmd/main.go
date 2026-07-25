package main

import (
	"os"

	"github.com/elecbug/crawlp/internal/app"
)

func main() {
	flags := app.ParseFlags()
	opts, err := app.ParseOptions(flags)
	if err != nil {
		app.PrintError(err)
		if opts.PauseOnFinish {
			app.WaitForEnter()
		}
		os.Exit(2)
	}

	err = app.Run(opts)
	if err != nil {
		app.PrintError(err)
		if opts.PauseOnFinish {
			app.WaitForEnter()
		}
		os.Exit(1)
	}

	if opts.PauseOnFinish {
		app.WaitForEnter()
	}
}
