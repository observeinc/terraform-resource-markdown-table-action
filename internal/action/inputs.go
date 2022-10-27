package action

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var ErrNoResources = errors.New("no resources defined")

type Inputs struct {
	WorkingDirectory string
	Resources        ResourcesInput
	OutputFile       string
}

type ResourcesInput string

func (r ResourcesInput) Parse() (Resources, error) {
	rs := Resources{}

	if err := yaml.Unmarshal([]byte(r), &rs); err != nil {
		return nil, err
	}

	return rs, nil
}

type Resources []*Resource

func (r Resources) Validate() error {
	if len(r) == 0 {
		return ErrNoResources
	}

	for _, resource := range r {
		if err := resource.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type Resource struct {
	Name       string   `yaml:"name"`
	Attributes []string `yaml:"attributes"`
}

func (r *Resource) Validate() error {
	if len(r.Attributes) == 0 {
		return &NoResourceAttributesError{Name: r.Name}
	}

	return nil
}

type NoResourceAttributesError struct {
	Name string
}

func (e *NoResourceAttributesError) Error() string {
	return fmt.Sprintf("No attributes defined for resource %q", e.Name)
}
