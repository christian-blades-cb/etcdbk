# etcdbk
[![](https://badge.imagelayers.io/christianbladescb/etcdbk:latest.svg)](https://imagelayers.io/?images=christianbladescb/etcdbk:latest 'Get your own badge on imagelayers.io')

[![](http://dockeri.co/image/christianbladescb/etcdbk)](https://registry.hub.docker.com/u/christianbladescb/etcdbk)

Etcdbk is a tool for backing up the values in a live etcd cluster. 

Etcdbk generates a tarball (tar.gz) with paths matching the keys in your etcd cluster. Etcdbk uses the etcd API instead of direct access to the etcd data directory, thus allowing backups to be generated on machines remote to the target cluster, or from within application containers.

## Installation

```shell
$ go get github.com/christian-blades-cb/etcdbk
```

Alternatively, a docker image is available at [christianbladescb/etcdbk](https://registry.hub.docker.com/u/christianbladescb/etcdbk/), or can be built via the `build.sh` script in the root of this repository.

## Usage

etcdbk CLI has 3 commands to suit your usecase.

* `file` backs up the etcd database to a local file
* `s3` backs up the etcd database to an S3 bucket
* `s3 continuous` watches for changes to the etcd database, and backs up on set hard intervals, and set intervals after a change

### One-time backup to a local file

```
Usage:
  etcdbk [OPTIONS] file [file-OPTIONS]

Output a tarball representing the etcd database to a file on disk.

Application Options:
  -e, --etcd-hosts=   etcd machines (http://127.0.0.1:4001) [$ETCD_HOSTS]
  -v, --debug         verbose logging

Help Options:
  -h, --help          Show this help message

[file command options]
      -o, --outfile=  Where to write the resulting tarball (STDOUT if not set) [$OUTFILE]
```

#### Simple Example 

To backup the keys from a cluster with a node running on localhost:

```shell
$ etcdbk file -o ./my-etcd-backup.tar.gz
```

The archive at `./my-etcd-backup.tar.gz` will contain a file system corresponding to the keys available in the etcd cluster.

### One-time backup to S3

```
Usage:
  etcdbk [OPTIONS] s3 [s3-OPTIONS] <continuous>

Output a tarball representing an etcd database into an S3 bucket

Application Options:
  -e, --etcd-hosts=       etcd machines (http://127.0.0.1:4001) [$ETCD_HOSTS]
  -v, --debug             verbose logging

Help Options:
  -h, --help              Show this help message

[s3 command options]
      -n, --cluster-name= Cluster name to use in naming the file in the S3 Bucket (etcd-cluster) [$CLUSTER_NAME]
          --aws-access=   Access key of an IAM user with write access to the given bucket [$AWS_ACCESS_KEY_ID]
          --aws-secret=   Secret key of an IAM user with write access to the given bucket [$AWS_SECRET_ACCESS_KEY]
          --s3-endpoint=  AWS S3 endpoint. See http://goo.gl/OG2Nkv (https://s3.amazonaws.com) [$AWS_S3_ENDPOINT]
          --aws-bucket=   Bucket in which to place the archive. [$AWS_S3_BUCKET]

Available commands:
  continuous  Backup to S3 continuously
```

#### Example ####

Assuming you have previously created an **S3 bucket** and an IAM user with **write access** to that bucket:

```shell
$ etcdbk --cluster-name=my-etcd-cluster s3 --aws-access=ACCESSKEY --aws-secret=SECRETKEYSAREALWAYSLONGER --s3-endpoint=https://s3.amazonaws.com --aws-bucket=etcdbackups
```

An archive will be saved into the specified bucket. The archive name will be in the format `#{cluster name}-#{time in RFC3339}.tar.gz`.

### Continous backup to S3

```
Usage:
  etcdbk [OPTIONS] s3 [s3-OPTIONS] continuous [continuous-OPTIONS]

Backup an etcd database at regular intervals, or after changes

Application Options:
  -e, --etcd-hosts=       etcd machines (http://127.0.0.1:4001) [$ETCD_HOSTS]
  -v, --debug             verbose logging

Help Options:
  -h, --help              Show this help message

[s3 command options]

    Output to S3 bucket:
      -n, --cluster-name= Cluster name to use in naming the file in the S3 Bucket (etcd-cluster) [$CLUSTER_NAME]
          --aws-access=   Access key of an IAM user with write access to the given bucket [$AWS_ACCESS_KEY_ID]
          --aws-secret=   Secret key of an IAM user with write access to the given bucket [$AWS_SECRET_ACCESS_KEY]
          --s3-endpoint=  AWS S3 endpoint. See http://goo.gl/OG2Nkv (https://s3.amazonaws.com) [$AWS_S3_ENDPOINT]
          --aws-bucket=   Bucket in which to place the archive. [$AWS_S3_BUCKET]

[continuous command options]
          --max-period=   Longest time to wait between snapshots if there are no updates (168h) [$MAX_PERIOD]
          --min-period=   How long to wait after an update to push the snapshot to S3 (1h) [$MIN_PERIOD]
```

## Alternatives

* [etcdctl backup](https://github.com/coreos/etcd/blob/master/Documentation/admin_guide.md)
* [etcd-backup](https://github.com/fanhattan/etcd-backup)
