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
mise trust
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

## Running the Benchmark

## Analysis
