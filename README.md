# etcdbk
[![](https://badge.imagelayers.io/christianbladescb/etcdbk:latest.svg)](https://imagelayers.io/?images=christianbladescb/etcdbk:latest 'Get your own badge on imagelayers.io')

[![](http://dockeri.co/image/christianbladescb/etcdbk)](https://registry.hub.docker.com/u/christianbladescb/etcdbk)

Etcdbk is a tool for backing up the values in a live etcd cluster. 

Etcdbk generates a tarball (tar.gz) with paths matching the keys in your etcd cluster. Etcdbk uses the etcd API instead of direct access to the etcd data directory, thus allowing backups to be generated on machines remote to the target cluster, or from within application containers.

**Caveat:** Because etcdbk only adds keys to resulting artifact, etcd paths that have no keys will be lost.

## Installation

```shell
$ go get github.com/christian-blades-cb/etcdbk
```

Alternatively, a docker image is available at [christianbladescb/etcdbk](https://registry.hub.docker.com/u/christianbladescb/etcdbk/), or can be built via the `build.sh` script in the root of this repository.

## Usage

```
Usage:
  etcdbk [OPTIONS]

Application Options:
  -e, --etcd-hosts=   etcd machines (http://127.0.0.1:2379) [$ETCD_HOSTS]
  -n, --cluster-name= Cluster name to use in naming the file in the S3 Bucket (etcd-cluster) [$CLUSTER_NAME]
  -o, --outfile=      Where to write the resulting tarball. '-' for STDOUT [$OUTFILE]
      --aws-access=
      --aws-secret=
      --s3-endpoint=  AWS S3 endpoint. See http://goo.gl/OG2Nkv (https://s3.amazonaws.com) [$AWS_S3_ENDPOINT]
      --aws-bucket=

Help Options:
  -h, --help          Show this help message
```

### Simple Example 

To backup the keys from a cluster with a node running on localhost:

```shell
$ etcdbk -o ./my-etcd-backup.tar.gz
```

The archive at `./my-etcd-backup.tar.gz` will contain a file system corresponding to the keys available in the etcd cluster.

### Backing up to S3

Assuming you have previously created an **S3 bucket** and an IAM user with **write access** to that bucket:

```shell
$ etcdbk --cluster-name=my-etcd-cluster --aws-access=ACCESSKEY --aws-secret=SECRETKEYSAREALWAYSLONGER --s3-endpoint=https://s3.amazonaws.com --aws-bucket=etcdbackups
```

An archive will be saved into the specified bucket. The archive name will be in the format `#{cluster name}-#{time in RFC3339}.tar.gz`.

## Alternatives

* [etcdctl backup](https://github.com/coreos/etcd/blob/master/Documentation/admin_guide.md)
* [etcd-backup](https://github.com/fanhattan/etcd-backup)
