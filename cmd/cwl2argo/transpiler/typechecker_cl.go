package transpiler

import (
	"errors"
	"fmt"
)

func errorNilRequirements(id *string) error {
	if id != nil {
		return fmt.Errorf("Requirements cannot be nil in %s", *id)
	} else {
		return errors.New("Requiremnets cannot be nil")
	}
}

func errorDockerRequirement(id *string) error {
	if id != nil {
		return fmt.Errorf("DockerRequirement must be present in all Argo CWL definitions, %s does not satisfy this", *id)
	} else {
		return errors.New("DockerRequirement must be present in all Argo CWL definitions")
	}
}

func TypeCheckCommandlineInputs(clins []CommandlineInputParameter) error {
	for _, clin := range clins {
		// type check secondary files
		if clin.SecondaryFiles != nil {
		}
	}
	return nil
}

func TypeCheckCommandlineOutputs(clouts []CommandlineOutputParameter) error {

	return nil
}

func TypeCheckCommandlineClass(id *string, class string) error {
	if class == "CommandlineTool" {
		return nil
	}
	if id != nil {
		return fmt.Errorf("\"CommandlineTool\" required but %s was provided in %s", class, *id)
	} else {
		return fmt.Errorf("\"CommandlineTool\" required but %s provided", class)
	}
}

func TypeCheckCommandlineRequirements(id *string, clrs []CWLRequirements) error {
	if clrs == nil {
		return errorNilRequirements(id)
	}

	foundDocker := false

	for _, requirement := range clrs {
		if _, ok := requirement.(DockerRequirement); ok == true {
			foundDocker = true
			break
		}
	}

	if foundDocker == false {
		return errorDockerRequirement(id)
	}
	return nil
}

func TypeCheckCommandlineHints(id *string, hints []interface{}) error {

	return nil
}

func TypeCheckCLICWLVersion(id *string, cwlVersion *string) error {
	// allowed to be nil
	if cwlVersion == nil {
		return nil
	}
	if cwlVersion != nil && *cwlVersion == CWLVersion {
		return nil
	}
	if id != nil {
		return fmt.Errorf("In %s cwlVerion provided was %s but %s was expected", *id, *cwlVersion, CWLVersion)
	} else {
		return fmt.Errorf("cwlVersion provided was %s but %s was expected", *cwlVersion, CWLVersion)
	}
}

func TypeCheckBaseCommand(id *string, baseCommand []string, arguments []CommandlineArgument) error {

	if len(baseCommand) > 0 || len(arguments) > 0 {
		return nil
	}
	if id != nil {
		return fmt.Errorf("In %s len(baseCommand) == 0 and len(arguments) was not > 0", *id)
	} else {
		return errors.New("If len(baseCommand) == 0 then len(arguments) must be > 0")
	}
}

func TypeCheckCommandlineTool(cl *CommandlineTool, inputs map[string]interface{}) error {
	var err error

	err = TypeCheckCommandlineInputs(cl.Inputs)
	if err != nil {
		return err
	}

	err = TypeCheckCommandlineOutputs(cl.Outputs)
	if err != nil {
		return err
	}

	err = TypeCheckCommandlineClass(cl.Id, cl.Class)
	if err != nil {
		return err
	}

	err = TypeCheckCommandlineRequirements(cl.Id, cl.Requirements)
	if err != nil {
		return err
	}

	err = TypeCheckCommandlineHints(cl.Id, cl.Hints)
	if err != nil {
		return err
	}

	err = TypeCheckCLICWLVersion(cl.Id, cl.CWLVersion)
	if err != nil {
		return nil
	}

	err = TypeCheckBaseCommand(cl.Id, cl.BaseCommand, cl.Arguments)
	if err != nil {
		return err
	}

	return nil
}
