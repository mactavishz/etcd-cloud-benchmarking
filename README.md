# CSB Project WS2425

This repository contains the code for the benchmark implementaion, benchmark raw data, analysis scripts, generated figures etc.

## Getting started

The following instructions will help you set up the project on your local machine and enable you to run the benchmark.

### Prerequisites

To run the project you will need to have the following software installed on your machine:

- [Python 3.12.*](https://www.python.org/downloads/) for running the analysis scripts
- [Go 1.23.*](https://go.dev/doc/install) for the benchmark implementation
- [Etcd 3.5.*](https://etcd.io/docs/v3.5/install/) for local testing
- [Make](https://www.gnu.org/software/make/) for running the build and test commands
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) for provisioning resources on Google Cloud Platform

We recommend you to use [mise](https://mise.jdx.dev/getting-started.html) to manage your go and python environments if you have multiple different versions of go and python installed on your machine. We have provided the `.mise.toml` file to help you set up the environment.

If you are using MacOS on M1 chip, make sure your python is native to arm64 architecture (not using Rosetta). You can check this by running `python -c "import platform; print(platform.platform())"`. Make sure you the output contains `arm64`. This is important for running the analysis scripts later.

### Setting up Environment

If you are using mise, you can set up the environment by running the following command in the root directory of the project:

```bash
# trust the mise.toml file
mise trust

# install the tools defined in the configuration file
mise install
```

First, install the go dependencies by running the following command in the root directory of the project:

```bash
go work sync
```
After that, you can try to run the following command to build the benchmark-related binaries:

```bash
make

# clean up the binaries
make clean
```

Second, create the virtual environment for python and install the python dependencies by running the following command in the root directory of the project:

```bash
# create the virtual environment
python -m venv .venv

# activate the virtual environment on MacOS/Linux
source .venv/bin/activate

# verify the virtual environment is activated
which python

# install the python dependencies
pip install -r requirements.txt
```

Third, set up the gcloud CLI by running the following command:

```bash
gcloud init
```

## Local Development and Testing

The benchmark program consists of 4 components, all written in Go:

- `control`: the control program to start the benchmark on your local machine, communicates with the benchmark client program
- `client`: the benchmark client program to send requests to the etcd cluster and record the metrics
- `api`: protobuf definitions for the communication between the control program and the benchmark client program
- `data-generator`: the data generator program to generate the synthetic data for the benchmark

These are 4 different go modules, only the `control` and `client` are compiled into binaries.

Here are some make commands to help you build and test the components:

```bash
# build the control and client binaries
make all

# generate protobuf go code
make gen

# clean up
make clean

# run unit tests
make test
```

After compiling the binaries, two binaries will be generated in the `bin` directory:

- `./bin/benchctl`
- `./bin/benchclient`

For `benchctl`, there are several subcommands available:

```bash
# display the help message
./bin/benchctl help

# display the help message for the config subcommand
./bin/benchctl config -h

# display the help message for the run subcommand
./bin/benchctl run -h
```

For `benchclient`, there are only one flag available:

```bash
# display the help message
./bin/benchclient -h
````

### Running the Benchmark Locally

First, you have to initialize the etcd cluster on your local machine. You can run the following command in a separate terminal session:

```bash
etcd
```

Then, you have to initialize the configuration for the benchmark, you can run the following command:

```bash
# initialize the configuration
./bin/benchctl config init

# or load the configuration from a file
./bin/benchctl config load-file ./some-config-file.json
```

You can view and change the configuration by running the following command:

```bash
# view the content of the configuration in JSON format
./bin/benchctl config view

# view the content of the configuration in table format
./bin/benchctl config list

# get a value of a field in the configuration
./bin/benchctl config get <field-name>

# set a value of a field in the configuration
./bin/benchctl config set <field-name>=<value>
```

Now you can run the client program first to wait for the control program to start the benchmark in a separate terminal session:

```bash
./bin/benchclient -p 50050
```

Then, you can run the control program to start the benchmark in a separate terminal session:

```bash
./bin/benchctl run 127.0.0.1:50050
```

Now the benchmark will start running, you can view the status messages in the terminal where the client/control program is running.

## Running the Benchmark

To run the benchmark, you first have to provision the etcd cluster and the benchmark client machine on Google Cloud Platform. We have provided the shell script to help you provision the resources. These scripts are located in the `infra` directory.

You can run the following command to display the help message:

```bash
# first 
cd ./infra

# chekc the help message
./provision.sh -h
```

The help message looks like this:

```text
Usage: ./provision.sh <command> [options]
Commands:
  deploy   - Deploy etcd cluster with number of nodes specified by -n, along with 1 benchmark machine
  cleanup  - Cleanup all resources
Options:
  -n, --num_nodes <number>  The number of nodes in the etcd cluster. Default: 1
  -z  --zone <zone>         The GCP zone in which the instances are deployed, available options: a, b, c. Default: c
  -y, --yes                 Skip confirmation prompt
  -h, --help                Print this help message
```

There are only two commands available: `deploy` and `cleanup`. The `deploy` command will provision the etcd cluster and the benchmark client machine, as well as install all the neccessary softwares to run the benchmark. The `cleanup` command will delete all the resources created by the `deploy` command.

Currently all the resources are deployed in the region `europe-central2`, however to prevent the shortage, we let you specify the zone in which the instances are deployed. The available options are `a`, `b`, and `c`. The default zone is `c`.

Make sure you first set up the gcloud CLI before running the script, including setting the default project and a billing account. The script will also prompt to to confirm your project, but you can skip this by passing the `-y` flag.

### Provision the etcd cluster and a benchmark client machine

Run the following command in side the `infra` directory: 

```text
# deploy 1 etcd node and 1 benchmark client machine
./provision.sh deploy -n 1

# deploy 3 etcd nodes and 1 benchmark client machine
./provision.sh deploy -n 3

# deploy 5 etcd nodes and 1 benchmark client machine
./provision.sh deploy -n 5
```

You need to wait for the script to finish. After the script finishes, you will see the IP addresses of all the instances and health check status of the etcd cluster.

Later, you can run the following command to clean up the resources:

```bash
./provision.sh cleanup
```

### Start the benchmark run

We have provided the shell script to help you start running the benchmark. The script is located in the `benchmark` directory.

You can run the following command to display the help message:

```bash
cd ./benchmark

./start.sh -h
```

You will see the help message like this:

```text
Usage: ./start.sh [options]
Options:
  -n, --num_nodes <number>  The number of nodes in the etcd cluster. Default: 1
  -z  --zone <zone>         The GCP zone in which the instances are deployed, available options: a, b, c. Default: c
  -s, --scenario <name>     The benchmark scenario name to run, available options: kv, lock. Default: kv
  -w  --workload <name>     The workload type in kv/lock scenario to run, available options for scenario kv: read-only, read-heavy, update-heavy, available options for scenario lock: lock-only, lock-mixed-read, lock-mixed-write, lock-contention. Multiple values are provided with commas in between, or all can be used for sepcifying all workloads. Default: all
  -d  --out_dir <path>      The output directory for the benchmark results, can be relative or absolute path. Default: results
  -h, --help                Print this help message
```

You can run the following command to start the benchmark, the value specified by the `-n` flag must be the same as the number of nodes in the etcd cluster you provisioned using the `provision.sh` script. If you have also specified a different zone in the `provision.sh` script, you have to specify the same zone using the `-z` flag:

```bash
# start the benchmark with 3-node etcd cluster and run all workloads in kv scenario
 ./start.sh -s kv -w all -n 3 -d results/run1/kv/3-node

# start the benchmark with 5-node etcd cluster and run all workloads in lock scenario
 ./start.sh -s lock -w all -n 5 -d results/run1/lock/5-node
```

You have to specify the output folder for the benchmark results using the `-d` flag. The benchmark results will be stored in the specified folder. We recommend you to use the following folder structure to store the benchmark results:

```text
run1
├── kv
│   ├── 1-node
│   ├── 3-node
│   └── 5-node
└── lock
    ├── 1-node
    ├── 3-node
    └── 5-node
```

This is the folder structure recognized by the analysis scripts. If you want to run all the scenarios and workloads, you can run the following command:

```bash
for n in 1 3 5; do
  ./start.sh -s kv -w all -n $n -d results/run-1/kv/$n-node
  ./start.sh -s lock -w all -n $n -d results/run-1/lock/$n-node
done
```
This will complete a whole round of benchmark runs for all the scenarios and workloads with different cluster sizes. If the benchmark runs finshes successfully, you will see the benchmark results with the following folder structure:

```text
run1
├── kv
│   ├── 1-node
│   │   ├── read-heavy
│   │   ├── read-only
│   │   └── update-heavy
│   ├── 3-node
│   │   ├── read-heavy
│   │   ├── read-only
│   │   └── update-heavy
│   └── 5-node
│       ├── read-heavy
│       ├── read-only
│       └── update-heavy
└── lock
    ├── 1-node
    │   ├── lock-contention
    │   ├── lock-mixed-read
    │   ├── lock-mixed-write
    │   └── lock-only
    ├── 3-node
    │   ├── lock-contention
    │   ├── lock-mixed-read
    │   ├── lock-mixed-write
    │   └── lock-only
    └── 5-node
        ├── lock-contention
        ├── lock-mixed-read
        ├── lock-mixed-write
        └── lock-only
```

Inside each workload folder, you will see the benchmark results in the form of CSV files along with the log files:

```text
run1
├── kv
│   ├── 1-node
│   │   ├── read-heavy
│   │   │   ├── CPU_Utilization.csv
│   │   │   ├── Memory_Utilization.csv
│   │   │   ├── keys.txt
│   │   │   ├── metrics.csv
│   │   │   └── run.log
...
```


Dont forget to clean up the resources after you finish the benchmark run, as the resources are billed by Google Cloud Platform.

## Analysis

After you have finished running the benchmark, you can run the analysis scripts to generate the figures. The analysis scripts are located in the `benchmark` directory.

Run the following command to analyze the benchmark results:

```bash
# suppose you have the benchmark results stored in the folder ./results/run1
python ./analysis.py --root=./results/run1 --output run1_analysis
```

This will generate the figures in the folder `run1_analysis`. The figures will be stored in the following folder structure:

```text
run1_analysis
├── distribution
├── error_rate
├── latency
├── scalability
├── system
└── throughput
```
