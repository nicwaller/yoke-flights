package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yokecd/yoke/pkg/flight"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	values := defaults

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		if err := yaml.NewDecoder(os.Stdin).Decode(&values); err != nil && err != io.EOF {
			return fmt.Errorf("failed to decode values: %w", err)
		}
	}

	resources, err := render(flight.Release(), flight.Namespace(), values)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(resources)
}
