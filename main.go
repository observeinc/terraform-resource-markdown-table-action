package main

import (
	"context"

	"github.com/observeinc/terraform-resource-markdown-table-action/internal/action"
	"github.com/sethvargo/go-githubactions"
)

func main() {
	inputs := action.Inputs{
		WorkingDirectory: githubactions.GetInput("working_directory"),
		OutputFile:       githubactions.GetInput("output_file"),
		ResourceTypes:    action.ResourcesInput(githubactions.GetInput("resources")),
	}

	if err := action.Run(context.Background(), inputs); err != nil {
		githubactions.Fatalf("%v", err)
	}
}
