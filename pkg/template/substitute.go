package template

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Substitute performs Docker-Compose-style variable substitution on the provided input string.
// It supports the following patterns that are compatible with the Compose specification:
//  1. ${VAR}                    – Substitute with value (empty string if unset)
//  2. ${VAR:-default}           – Default if VAR is unset OR empty
//  3. ${VAR-default}            – Default if VAR is unset (but *not* if it is set to an empty string)
//  4. ${VAR:?err}               – Error if VAR is unset OR empty (message "err")
//  5. ${VAR?err}                – Error if VAR is unset (message "err")
//
// The lookup order for variable values is:
//  1. Provided vars map (highest priority)
//  2. Operating-system environment variables
//
// The function returns the resulting string or an error if mandatory
// variables are missing as dictated by the interpolation rules.
func Substitute(input string, vars map[string]string) (string, error) {
	// Pre-compile regex once.  It captures the inside of the brace pair.
	// Example match: ${MY_VAR:-default}
	varPattern := regexp.MustCompile(`\$\{([^}]+)}`)

	// Short-circuit: if no pattern present, return input as-is.
	if !varPattern.MatchString(input) {
		return input, nil
	}

	type matchPos struct {
		fullStart int
		fullEnd   int
		exprStart int
		exprEnd   int
	}

	// Collect matches first so that we can build output in a single pass while
	// properly handling substitution errors.
	indices := varPattern.FindAllStringSubmatchIndex(input, -1)
	if len(indices) == 0 {
		return input, nil
	}

	// Builder for the final string – allocate roughly the same size as input.
	var builder strings.Builder
	builder.Grow(len(input))

	lastPos := 0
	for _, idx := range indices {
		m := matchPos{
			fullStart: idx[0],
			fullEnd:   idx[1],
			exprStart: idx[2],
			exprEnd:   idx[3],
		}

		// Write any text preceding this match.
		builder.WriteString(input[lastPos:m.fullStart])

		expr := input[m.exprStart:m.exprEnd]
		substitution, err := evaluateExpression(expr, vars)
		if err != nil {
			return "", err
		}
		builder.WriteString(substitution)

		lastPos = m.fullEnd
	}

	// Write the remainder of the string after the final match.
	builder.WriteString(input[lastPos:])

	return builder.String(), nil
}

// evaluateExpression processes a single variable expression (without the enclosing ${}).
// It follows the semantics described in the Substitute function header.
func evaluateExpression(expr string, vars map[string]string) (string, error) {
	// Determine the operator and split into name / suffix parts.
	var (
		name    string
		op      string
		operand string
	)

	// Helper to split by operator, checking for the two-character variants first.
	split := func(token string) (string, string, bool) {
		if idx := strings.Index(expr, token); idx != -1 {
			return expr[:idx], expr[idx+len(token):], true
		}
		return "", "", false
	}

	// Order matters – two-character tokens first.
	switch {
	case strings.Contains(expr, "::-"): // unlikely typo, but prevents \n confusion.
		// no-op – will fall through to default case.
	case strings.Contains(expr, ":-"):
		name, operand, _ = split(":-")
		op = ":-"
	case strings.Contains(expr, ":?"):
		name, operand, _ = split(":?")
		op = ":?"
	case strings.Contains(expr, "-"):
		name, operand, _ = split("-")
		op = "-"
	case strings.Contains(expr, "?"):
		name, operand, _ = split("?")
		op = "?"
	default:
		name = expr
		operand = ""
		op = "" // simple substitution
	}

	// Trim any accidental whitespace around the variable name.
	name = strings.TrimSpace(name)

	// Retrieve the variable value – precedence: vars map, then environment.
	value, exists := lookupVariable(name, vars)

	// Evaluate based on operator semantics.
	switch op {
	case "":
		if exists {
			return value, nil
		}
		return "", nil // Unset becomes an empty string.

	case "-": // default if UNSET
		if exists {
			return value, nil
		}
		return operand, nil

	case ":-": // default if UNSET or EMPTY
		if exists && value != "" {
			return value, nil
		}
		return operand, nil

	case "?": // error if UNSET
		if exists {
			return value, nil
		}
		return "", fmt.Errorf("variable %s is not set: %s", name, operand)

	case ":?": // error if UNSET or EMPTY
		if exists && value != "" {
			return value, nil
		}
		return "", fmt.Errorf("variable %s is not set or empty: %s", name, operand)

	default:
		// This should never happen, but handle gracefully.
		return "", fmt.Errorf("invalid variable expression: ${%s}", expr)
	}
}

// lookupVariable returns (value, exists) where exists indicates whether the variable was found.
func lookupVariable(name string, vars map[string]string) (string, bool) {
	if vars != nil {
		if v, ok := vars[name]; ok {
			return v, true
		}
	}
	v, ok := os.LookupEnv(name)
	return v, ok
}
