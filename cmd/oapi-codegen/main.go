package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/doordash/oapi-codegen/v3/pkg/codegen"
	"gopkg.in/yaml.v3"
)

var (
	flagConfigFile string
	flagOutputFile string
	flagPrintUsage bool
)

func main() {
	flag.StringVar(&flagConfigFile, "config", "", "A YAML config file that controls oapi-codegen behavior.")
	flag.StringVar(&flagOutputFile, "o", "", "Where to output generated code, stdout is default.")
	flag.BoolVar(&flagPrintUsage, "help", false, "Show this help and exit.")

	flag.Parse()

	if flagPrintUsage {
		flag.Usage()
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		errExit("Please specify a path to a OpenAPI 3.0 spec file")
	} else if flag.NArg() > 1 {
		errExit("Only one OpenAPI 3.0 spec file is accepted and it must be the last CLI argument")
	}

	// Read the spec file
	filepath := flag.Arg(0)
	specContents, err := os.ReadFile(filepath)
	if err != nil {
		errExit("Error reading file: %v", err)
	}

	// Read the config file
	cfgContents, err := os.ReadFile(flagConfigFile)
	if err != nil {
		errExit("Error reading config file: %v", err)
	}
	cfg := codegen.Configuration{}
	err = yaml.Unmarshal(cfgContents, &cfg)
	if err != nil {
		errExit("Error parsing config file: %v", err)
	}

	res, err := codegen.Generate(specContents, cfg)
	if err != nil {
		errExit("Error generating code: %v", err)
	}

	if flagOutputFile != "" {
		err = os.WriteFile(flagOutputFile, []byte(res), 0644)
		if err != nil {
			errExit("Error writing file: %v", err)
		}
	} else {
		println(res)
	}
}

func errExit(msg string, args ...any) {
	msg = msg + "\n"
	_, _ = fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
