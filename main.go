package main

import (
	"context"
	"strconv"

	"github.com/observeinc/terraform-resource-markdown-table-action/internal/action"
	"github.com/sethvargo/go-githubactions"
)

const defaultHeaderLevel = 2

func main() {
	inputs := action.Inputs{
		WorkingDirectory: githubactions.GetInput("working_directory"),
		OutputFile:       githubactions.GetInput("output_file"),
		ResourceTypes:    action.ResourcesInput(githubactions.GetInput("resources")),
		HeaderLevel:      headerLevelFromInput(githubactions.GetInput("resource_header_level")),
	}

	if err := action.Run(context.Background(), inputs); err != nil {
		githubactions.Fatalf("%v", err)
	}
}

func headerLevelFromInput(input string) int {
	if input == "" {
		return defaultHeaderLevel
	}

	i, err := strconv.Atoi(input)
	if err != nil {
		githubactions.Fatalf("failed to parse resource_header_level: %v", err)
	}

	return i
}
