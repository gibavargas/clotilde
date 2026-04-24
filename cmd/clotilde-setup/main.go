package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	var opts Options
	flag.BoolVar(&opts.NonInteractive, "non-interactive", false, "run without prompts; requires a config file or complete environment-derived config")
	flag.StringVar(&opts.ConfigPath, "config", "", "path to setup JSON config")
	flag.StringVar(&opts.Output, "output", "text", "output format: text or json")
	flag.BoolVar(&opts.Yes, "yes", false, "assume yes for confirmations")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "plan commands without changing Google Cloud resources")
	flag.BoolVar(&opts.SkipBuild, "skip-build", false, "skip docker build and push")
	flag.BoolVar(&opts.SkipVerify, "skip-verify", false, "skip post-deploy HTTP verification")
	flag.StringVar(&opts.ResultPath, "result-path", "", "path for sanitized setup result JSON")
	flag.Parse()

	if opts.Output != "text" && opts.Output != "json" {
		fmt.Fprintln(os.Stderr, "invalid --output; expected text or json")
		os.Exit(exitValidation)
	}

	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		result := SetupResult{OK: false, Error: err.Error()}
		if opts.Output == "json" {
			printJSON(os.Stdout, result)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitValidation)
	}
	applyDefaults(&cfg)

	var runner CommandRunner = &ShellRunner{}
	if opts.DryRun {
		runner = &DryRunRunner{
			ProjectID:   cfg.ProjectID,
			Region:      cfg.Region,
			ServiceName: cfg.ServiceName,
		}
	}

	result, code := runSetup(context.Background(), cfg, opts, runner, os.Stdin, os.Stdout, os.Stderr)
	if code != exitOK {
		if opts.Output == "json" {
			printJSON(os.Stdout, result)
		} else {
			fmt.Fprintln(os.Stderr, result.Error)
		}
	}
	os.Exit(code)
}
