# Hash Enrichment Plugin

This action enriches SysFlow records with a pre-computed list of hashes stored in a CSV file.

## Pre-requisites

* Docker

## Configuration

The list of hashes is stored in `hashes.csv`, a comma-separated list of md5-path pairs (one per line).

An example policy is provided in `pipeline.hash.json`, in which the policy engine is congigured in `enrich` mode and configured with a general-purposed policy that calls into the `hash` action.

```json
{
  "processor": "policyengine",
  "in": "flat flattenerchan",
  "out": "evt eventchan",
  "policies": "../plugins/actions/hash/policy.yaml", 
  "mode": "enrich"     
},
```

> **Note** the field `policies` accepts a directory that may contain one or more yaml files.

## Build

First, change to the root of this repository. To build the plugin (set $TAG to a SysFlow release (>=`0.4.0`), `edge`, or `dev`):

```bash
docker run --rm \
    -v $(pwd)/plugins:/go/src/github.com/sysflow-telemetry/sf-processor/plugins \
    -v $(pwd)/resources:/go/src/github.com/sysflow-telemetry/sf-processor/resources \
    sysflowtelemetry/plugin-builder:$TAG \
    make -C /go/src/github.com/sysflow-telemetry/sf-processor/plugins/actions/hash
```

## Running

To test it, run the pre-built processor with the hash configuration and trace path (the path can a path to a trace file or a directory containing sysflow trace files).

```bash
docker run --rm \
    -v $(pwd)/plugins:/usr/local/sysflow/plugins \
    -v $(pwd)/resources:/usr/local/sysflow/resources \
    -w /usr/local/sysflow/bin \
    --entrypoint=/usr/local/sysflow/bin/sfprocessor \
    sysflowtelemetry/sf-processor:$TAG \
    -log=quiet -config=../plugins/actions/hash/pipeline.hash.json ../resources/traces/
```

In the output, observe that all records matching the policy speficied in `policy.yaml` are tagged by action `hash` with the md5 hashes of the executable and/or file resource. For example:

```json
{
  "@timestamp": "2019-03-25T19:48:02.831501974Z",
  "agent": {
    "type": "SysFlow",
    "version": "0.4.0"
  },
  "ecs": {
    "version": "1.7.0"
  },
  "event": {
    "action": "process-start",
    "category": "process",
    "duration": 0,
    "end": "2019-03-25T19:48:02.831501974Z",
    "kind": "event",
    "reason": "Hash enrichment",
    "severity": 0,
    "sf_ret": 0,
    "sf_type": "PE",
    "start": "2019-03-25T19:48:02.831501974Z",
    "type": "start"
  },
  "container": null,
  "process": {
    "args": "",
    "command_line": "./client ",
    "executable": "./client",
    "name": "client",
    "parent": {
      "args": "",
      "command_line": "/usr/bin/bash",
      "executable": "/usr/bin/bash",
      "name": "bash",
      "pid": 7130,
      "start": "1970-01-01T00:00:00Z"
    },
    "pid": 13824,
    "start": "1970-01-01T00:00:00Z",
    "thread": {
      "id": 13824
    }
  },
  "user": {
    "group": {
      "id": -1
    },
    "id": -1
  },
  "tags": [
    "process_md5_hash:3577fc17c1b964af7cfe2c17c73f84f3"
  ]
}
```
