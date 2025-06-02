package cli_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mhsantos/rlcp/cmd/cli"
)

func TestArguments(t *testing.T) {
	tcs := []struct {
		name           string
		args           []string
		expectedOption cli.Option
		expectedError  error
	}{
		{
			name: "valid help command",
			args: []string{"rlcp", "--help"},
			expectedOption: cli.Option{
				Op: cli.Help,
			},
		},
		{
			name:           "invalid help command",
			args:           []string{"rlcp", "-help"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid option: -help"),
		},
		{
			name:           "unexisting option",
			args:           []string{"rlcp", "kill", "process"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid option: kill"),
		},
		{
			name:           "invalid number of arguments",
			args:           []string{"rlcp", "duck", "and", "cover"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid command"),
		},
		{
			name: "valid run command with single argument",
			args: []string{"rlcp", "run", "pwd"},
			expectedOption: cli.Option{
				Op:   cli.Run,
				Args: []string{"pwd"},
			},
		},
		{
			name: "valid run command with multiple arguments",
			args: []string{"rlcp", "run", "ls -la ../"},
			expectedOption: cli.Option{
				Op:   cli.Run,
				Args: []string{"ls", "-la", "../"},
			},
		},
		{
			name: "valid run command with 1 level of nested args",
			args: []string{"rlcp", "run", "echo 'my name is jonas'"},
			expectedOption: cli.Option{
				Op:   cli.Run,
				Args: []string{"echo", "my name is jonas"},
			},
		},
		{
			name: "valid run command with 2 levels of nested args",
			args: []string{"rlcp", "run", "sh -c \"echo 'my name is jonas'\""},
			expectedOption: cli.Option{
				Op:   cli.Run,
				Args: []string{"sh", "-c", "echo 'my name is jonas'"},
			},
		},
		{
			name: "valid status command",
			args: []string{"rlcp", "status", "af1f8215-bee7-455d-874a-55f0e3fb20b5"},
			expectedOption: cli.Option{
				Op:   cli.Status,
				Args: []string{"af1f8215-bee7-455d-874a-55f0e3fb20b5"},
			},
		},
		{
			name:           "invalid status command argument",
			args:           []string{"rlcp", "status", "invalid-uuid"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid job id"),
		},
		{
			name: "valid output command",
			args: []string{"rlcp", "output", "6c4bc197-b0e4-4ee3-a6be-b3b591ffad70"},
			expectedOption: cli.Option{
				Op:   cli.Output,
				Args: []string{"6c4bc197-b0e4-4ee3-a6be-b3b591ffad70"},
			},
		},
		{
			name:           "invalid output command argument",
			args:           []string{"rlcp", "output", "invalid-uuid"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid job id"),
		},
		{
			name: "valid stop command",
			args: []string{"rlcp", "stop", "cc430a1e-ab90-4cc0-b3b5-0ed22303b99a"},
			expectedOption: cli.Option{
				Op:   cli.Stop,
				Args: []string{"cc430a1e-ab90-4cc0-b3b5-0ed22303b99a"},
			},
		},
		{
			name:           "invalid stop command argument",
			args:           []string{"rlcp", "stop", "invalid-uuid"},
			expectedOption: cli.Option{},
			expectedError:  cli.NewErrInvalidCommand("invalid job id"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			option, err := cli.ParseCommand(tc.args)
			if !cmp.Equal(tc.expectedOption, option) {
				t.Fatalf("Unexpected Option returned. Expected: %v, Actual: %v", tc.expectedOption, option)
			}
			if !errors.Is(err, tc.expectedError) {
				t.Fatal("invalid error returned")
			}
		})
	}

}
