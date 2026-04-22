# Tail command

This example simulates a job for a command that reads an ever growing log file with multiple clients listening to the job output.

The `addlog.sh` shell script can be used to append entries to a pseudo log file at a rate of 100KB per second.

To run this exmaple, you may use multiple terminal windows. Later on we will include instructions to run this example with docker containters.

Steps to simulate a tail command parsing a log file.

1. to compile the files. On the root of this project:
```
mkdir bin
go build -o bin/rlcp ./cmd/rlcp
go build -o bin/server ./cmd/server 
```

2. start adding logs to a pseudo server log file:
```
./examples/tail/addlog.sh > server.log
```

3. from a new terminal window start the rlcp server:
```
./bin/server
```

4. from a third window, run the CLI to get start a command to listen to the output:
```
./bin/rlcp run "tail -f -n +1 server.log"
````

5. copy the job id from the previous step and run:
```
./bin/rlcp output <job_id> > output1.txt
```

6. open a new ternial window and repeat the previous step:
```
./bin/rlcp output <job_id> > output2.txt
```

7. repeat the previous step on new windows as many times as you want.