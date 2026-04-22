# RLCP - Remote Linux Command Processor

RLCP is a command line tool to run a command in a remote Linux server.

When a client issues a command, a command id is returned. That id can be used later to retrieve the status and output from the command execution.

Having the command issuing detached from its output and status allows it to be queried by multiple clients. The command output and status is kept on the server in memory, so it's accessible as long as the server is up.

Usage:

```
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
        rlcp stop af1f8215-bee7-455d-874a-55f0e3fb20b5
```

## Security

RLCP uses mTLS to encrypt the communication between the client and the server. Details on how to setup the keys are coming soon.

## Communication

RLCP uses [gRPC](https://grpc.io/) to communicate with the server. The implementation is in the [internal/pb](cmd/internal/pb) package.

## Development

To build the client:
```
go build -o bin/rlcp ./cmd/rlcp
```

To build the server:
```
go build -o bin/server ./cmd/server
```

Running tests:
```
go test ./...
```
