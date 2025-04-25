# local development

# unit tests

To run tests:
```
go test ./...
```

To run linting:
```
sudo docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/:/root/.cache -w /app golangci/golangci-lint:latest golangci-lint run
```

# sam local testing

SAM local testing is done by creating local resources (dynamodb, api gateway, lambda). Dynamodb is set up with docker 
and a local dynamodb image. Tables need to get created every time, which is automated with a bash script. Sam-cli is 
used to simulate api gateway and lambda.

For testing prayertexter binary only, we deviate from the actual prayertexter production architecture. Sam local uses
local api gateway to test with prayertexter instead of an SNS topic event. The reason for this is that api gateway is
much easier to test locally. Because of this, it is using a separate cloudformation template as well as main.go file.
Both of these changes can be seen in dev/prayertexter.

Prerequisite software installs:

1. docker
2. make
3. aws-cli
4. sam-cli

Log into aws-cli with dummy access key and ID (this is required for creating local dynamodb tables):

```
aws configure
AWS Access Key ID [None]: 1
AWS Secret Access Key [None]: 1
Default region name [us-west-1]: 
Default output format [json]:
```

Compile:

There are several different options when compiling. To run local tests, you have to compile and have a bootstrap binary
available for sam local.

1. Compile prayertexter app with default x64 cpu architecture:
```
make build -C dev/
```
2. Compile prayertexter app with arm64 cpu architecture:
```
make build -C dev/ ARCH=arm64
```
3. Compile other-than prayertexter binary on arm64 cpu architecture:
```
make build -C dev/ APP=statecontroller ARCH=arm64
```
4. Delete /bin and bootstrap binary in current dir:
```
make -C dev/ clean
```

Compiled binary called bootstrap gets created in /dev/bin and then copied to where it needs to be for cloudformation to
pick it up.

Create ddb tables - these are needed for testing all applications (prayertexter, statecontroller, etc):

```
./dev/dynamodb/setup.sh
```

Test prayertexter:

1. compile binary with make command so it is up to date
2. Run command in terminal 1:
```
sudo sam local start-api --docker-network sam-backend -t dev/prayertexter/template.yaml
```
3. Run command in terminal 2:
```
curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"originationNumber":"+17777777777", "messageBody": "pray"}'
```
4. monitor sam local api logs to view text message response

Test statecontroller:

1. compile binary with make command so it is up to date
2. Run command:
```
sudo sam local invoke StateController -t deploy/statecontroller/template.yaml --docker-network sam-backend
```

Good dynamodb commands:
```
aws dynamodb list-tables --endpoint-url http://localhost:8000
```
```
for table in ActivePrayer General Member QueuedPrayer; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done
```

Troubleshooting:

If this error is seen:
```
samcli.commands.local.cli_common.user_exceptions.ImageBuildException: Error building docker image: The command '/bin/sh -c mv   
/var/rapid/aws-lambda-rie-x86_64 /var/rapid/aws-lambda-rie && chmod +x /var/rapid/aws-lambda-rie' returned a non-zero code: 255 
```
1. check cloudformation template - Resources - <lambda function name> - Properties - Architectures
2. if developers computer is x64, then comment out architecture lines
3. if developers computer is arm64 (newer macbook), then include architecture lines

If this error is seen:
```
Invoke failed error=fork/exec /var/task/bootstrap: no such file or directory
```
1. compile specific binary with make command, this error means binary is not in the proper place