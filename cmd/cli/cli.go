// Package cli has methods to validate the command line input and return the appropriate operation
// to be executed by the client.
//
// When invalid arguments are provided, an error message with details on why the command is invalid is returned.
// That error message also includes the cli helper message, which can also is shown when the option --help is used.

package cli

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// HelpPrompt displays the helper message with all the options accepted by rlcp
const HelpPrompt = `
rlcp - Remote Linux Command Processor

Usage: rlcp operation argument
   
OPERATIONS
    --help
        shows this prompt

    run <command>
        runs the informed <command> on the server. <command> should be a single word or if multiple words, encapsulated by double quotes.
        this command returns a job id to be used to either query the status, get the output or stop the job later.

        Examples:
         rlcp run pwd
         rlcp run "ls -la"
         rlcp run "tail -f server.log"
    
    status <job id>
        gets the status for the job or an error message if the id is invalid or the user doesn't have the appropriate permissions.

        Example:
        rlcp status bf7a1eae-8d25-4de5-995b-8c4d3ef8b848
    
    output <job id>
        prints the output for the job or an error message if the id is invalid or the user doesn't have the appropriate permissions.

        Example:
        rlcp output 8060271e-b776-4444-9e75-bd2e3db3cc7d

    stop <job id>
        stops the job identified by job id. Returns an error message if the id is invalid or the user doesn't have the appropriate permissions.

        Example:
        rlcp stop af1f8215-bee7-455d-874a-55f0e3fb20b5`

type Operation uint

const (
	Run Operation = iota
	Status
	Output
	Stop
	Help
)

type ErrInvalidCommand struct {
	err string
}

func NewErrInvalidCommand(err string) ErrInvalidCommand {
	return ErrInvalidCommand{err}
}

func (e ErrInvalidCommand) Error() string {
	return e.err
}

type Option struct {
	Op   Operation
	Args []string
}

func ParseCommand(args []string) (Option, error) {
	if len(args) == 1 {
		return Option{}, ErrInvalidCommand{"invalid command"}
	}

	if len(args) == 2 {
		if args[1] != "--help" {
			return Option{}, ErrInvalidCommand{fmt.Sprintf("invalid option: %s", args[1])}
		}
		return Option{
			Op: Help,
		}, nil
	}

	if len(args) == 3 {
		switch args[1] {
		case "run":
			runArgs := splitArguments(args[2])
			return Option{
				Op:   Run,
				Args: runArgs,
			}, nil
		case "status":
			return validateOperation(Status, args[2])
		case "output":
			return validateOperation(Output, args[2])
		case "stop":
			return validateOperation(Stop, args[2])
		default:
			return Option{}, ErrInvalidCommand{fmt.Sprintf("invalid option: %s", args[1])}
		}
	}
	return Option{}, NewErrInvalidCommand("invalid command")
}

// splitArguments leverages the OS parsing on the input, which guarantees that all quotes are balanced.
// it then constructs a stack to parse nested quotes and only return one argument per outer quote boundary, including any possible inner quotes.
func splitArguments(cmd string) []string {
	cmd = strings.Trim(cmd, " ")

	quoteStack := make([]rune, 0)

	args := make([]string, 0)
	arg := make([]rune, 0)

	for _, r := range cmd {
		switch r {
		case ' ':
			if len(quoteStack) == 0 {
				if len(arg) > 0 {
					args = append(args, string(arg))
					arg = make([]rune, 0)
				}
				continue
			}
			arg = append(arg, r)
		case '\\':
			if len(quoteStack) > 1 {
				arg = append(arg, r)
			}
		case '\'', '"':
			if len(quoteStack) == 1 && quoteStack[0] == r {
				quoteStack = make([]rune, 0)
				args = append(args, string(arg))
				arg = make([]rune, 0)
			} else {
				if len(quoteStack) > 0 {
					arg = append(arg, r)
					if quoteStack[len(quoteStack)-1] == r {
						quoteStack = quoteStack[:len(quoteStack)-1]
					} else {
						quoteStack = append(quoteStack, r)
					}
				} else {
					quoteStack = append(quoteStack, r)
				}
			}
		default:
			arg = append(arg, r)
		}
	}
	if len(arg) > 0 {
		args = append(args, string(arg))
	}
	return args
}

// validateOperation returns an Option object with the informed operation if the id parameter is a
// valid uuid, otherwise returns an error
func validateOperation(op Operation, id string) (Option, error) {
	_, err := uuid.Parse(id)
	if err != nil {
		return Option{}, NewErrInvalidCommand("invalid job id")
	}
	return Option{
		Op:   op,
		Args: []string{id},
	}, nil
}
