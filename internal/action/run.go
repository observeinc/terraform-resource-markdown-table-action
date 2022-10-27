package action

func Run(inputs Inputs) error {
	i := Inputs{
		Resources: "foo",
	}

	i.WorkingDirectory = "bar"

	return nil
}
