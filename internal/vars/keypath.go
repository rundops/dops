package vars

import "fmt"

// VarKeyPath builds the dotted key path used to store a parameter value
// in the vault, based on the parameter's scope.
func VarKeyPath(scope, paramName, catalogName, runbookName string) string {
	switch scope {
	case "catalog":
		return fmt.Sprintf("vars.catalog.%s.%s", catalogName, paramName)
	case "runbook":
		return fmt.Sprintf("vars.catalog.%s.runbooks.%s.%s", catalogName, runbookName, paramName)
	default:
		return fmt.Sprintf("vars.global.%s", paramName)
	}
}
