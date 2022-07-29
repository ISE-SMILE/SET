# SET - the Serverless Evaluation Toolkit

This project combines experiences from prior serverless client-side experimentation into a single file-driven benchmark driver.

## Workloads

We support multiple types of function workloads each with configurable complexity:

| Name  | Details                                                                                                                                                                                                                                     | Configuration                                                                  |
|-------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------|
| Prime | Uses the Miller Rabin Algorithm to test a random number                                                                                                                                                                                     | Size of the number                                                             | 
| Idle  | Simple function that sleeps for a given length.                                                                                                                                                                                             | Sleep length                                                                   |
| IO    | Function that randomly reads/writes/lists data from an S3-like API                                                                                                                                                                          | Read/Write distribution, ChunkSize per Operation, Iterations, Number of Inputs |
| Lloyd | Function that generates memory/cpu stress on system by performing low level array operations. Inspired by [Serverless Computing: An Investigation of Factors Influencing Microservice Performance](https://doi.org/10.1109/IC2E.2018.00039) | complexity level                                                               | 
| Lloyd | Parallel running Lyod function                                                                                                                                                                                                              | parallelism, synchronization                                                   |

We pre-defined 9 levels of complexity that configure each function from low to high stress, see [types.go](set/types.go).

Set uses a phase workload, defined by three parameters:
 - starting requests per second (warmup)
 - scaling factor (scale)
 - phase duration (length)

Each execution starts by sending `warmup` requests per second for `length` time. Followed by sending gradually increasing requests per seconds reaching `warmup*scale` within `lenght` time.
Followed by `warmup*scale` requests per second for `2*length`.

During the scaling phase, a use can also perform operational tasks such as configuring memory or redeploying code.

## Deployment
We support deployments on AWS, OpenWhisk. Planned for Google, Azure, IBM (pull requests welcome!).
We use Makefiles to automated deployments, ensure that `make`, `bash` and other unix tools are available.

For AWS we need the **sls** utility, configured with fitting access right.
For OW we need the **wsk** utility, configured with fitting access right.

## Usage
Set uses a file driven approach, thus, all experimenters are based on config files, to ensure reproducibility, see [Examples](example/).

```yaml
---
name: <exoermient name> #manditory 
target: <endpoint/function name> #optinoal
threads: 6 #threads used to send results (do not over commit local cpu resources to ensure realistic measurements)
warmup: 30 # target requests per second during warmup 
scaling: 1.5 # scaling factor 
phaseLength: 120s # duration of each phase
type: prime # workload function time
complexity: 1 # complexirt level (see types.go)
invoker: 
  type: ow # depends on th edeployment type, use http for AWS and OW for openwhisk
deployment:
  source: functions/ow/go # we provide a set of predefined deployment packages in functions/
  runtime: go:1.15 # Runtime name (depends on platform)
  memory: 512
  timeout: 90s
```

To use a config file run `set --workload <filename>`. All results are stored in the [data](data/) folder. 
We use the [faas-fact](https://github.com/faas-facts) library to collect metrics.
