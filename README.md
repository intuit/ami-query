# ami-query

Provide a RESTful API to query information about Amazon AWS AMIs.

## Prerequisites

`ami-query` is written in Go. You need to have version 1.5.1 or higher
installed. For Go installation instructions see https://golang.org/doc/install.

Third party packages are
[vendored](https://code.google.com/p/go-wiki/wiki/PackageManagementTools) using
the Go 1.5 [vendor experiment](https://golang.org/s/go15vendor).

## Installation

To install `ami-query` export `GOPATH` and `GO15VENDOREXPERIMENT` then use
`go get`.

```shell
$ export GOPATH=$HOME/go
$ export GO15VENDOREXPERIMENT=1
$ go get github.com/intuit/ami-query
```

`ami-query` will be installed into `$GOPATH/bin`. If you'd like to install it
into another directory (e.g. `/usr/local/bin`), then you can export `GOBIN`
prior to running `go get`.

To build an RPM for either RHEL 6 or 7, which can be used for installation on
other systems, `cd` into the source directory and run `make rpm`.

```shell
$ cd $GOPATH/src/github.com/intuit/ami-query
$ make rpm
```

## Configuration

The configuration is handled through the following environment variables.

### Required Values

* **AWS_ACCESS_KEY_ID**

  The access key used to communicate with AWS. If not found in the environment,
  `ami-query` will look for the `~/.aws/credentials` file or in the EC2 instance
  meta-data for an IAM role, in that order.

* **AWS_SECRET_ACCESS_KEY**

  The secret key used to communicate with AWS. If not found in the environment,
  `ami-query` will look in the `~/.aws/credentials` file or in the EC2 instance
  meta-data for an IAM role, in that order.

* **AMIQUERY_OWNER_IDS**

  A list of Owner IDs that created the AMIs. This is used to filter the AMI
  results.

### Optional Values

* **AMIQUERY_LISTEN_ADDRESS**

  The bind address and port that `ami-query` will listen on. The format for this
  value is `ip_addr:port` or `hostname:port`. The default value is
  `localhost:8080`.

* **AMIQUERY_REGIONS**

  A comma-separated list of regions that `ami-query` will scan for AMIs. Use
  "us-east-1" for US East, "us-west-1" for US West 1, etc.

* **AMIQUERY_ROLE_ARN**

  The AWS Resource Name of the role `ami-query` will assume (STS AssumeRole),
  e.g. `arn:aws:iam::123456789012:role/demo`. This can be used for running
  `ami-query` with a user/iam role outside of the account where the AMIs are
  built. Note that this requires the user/iam role to be granted permission from
  within the builder account.

* **AMIQUERY_CACHE_MANAGER**

  The type of cache to use. Currently supported values are `internal` and
  `memcached`. The default value is `internal`. See [Cache Types](#cachetypes)
  for more information.

* **AMIQUERY_CACHE_TTL**

  The time to wait before the AMI cache is updated. The format of this value is
  a duration such as "5s" or "5m". The default value is "15m", or 15 minutes.

* **AMIQUERY_MEMCACHED_SERVERS**

  A comma-separated list of memcached servers to use. This only needs to be
  defined when **AMIQUERY_CACHE_MANAGER** is set to `memcached`.

* **AMIQUERY_APP_LOGFILE**

   The file location to send application log messages. Note that `ami-query`
   does not manage this file, it only writes to it. The default is to log to
   STDERR.

* **AMIQUERY_HTTP_LOGFILE**

  The file location to send HTTP log messages. Note that `ami-query` does not
  manage this file, it only writes to it. The default is to log to STDERR.

#### Example Configuration

This most basic configuration listens on `localhost:8080` and uses the
`internal` cache manager with a 15 minute cache TTL. It will attempt to use
AWS credentials from either environment variables, `~/.aws/credentials`, or the
meta-data on an EC2 instance.

```shell
export AMIQUERY_OWNER_IDS="111122223333"
export AMIQUERY_REGIONS="us-east-1,us-west-2"
```

This configuration listens on localhost:8081 and uses the `memcached` cache
manager talking to `localhost:11211` and `localhost:11212`. It also sets the
cache TTL to 5 minutes.

```shell
export AMIQUERY_LISTEN_ADDRESS="localhost:8081"
export AMIQUERY_OWNER_IDS="111122223333"
export AMIQUERY_REGIONS="us-east-1,us-west-1"
export AMIQUERY_CACHE_MANAGER="memcached"
export AMIQUERY_MEMCACHED_SERVERS="localhost:112211,localhost:11212"
export AMIQUERY_CACHE_TTL="5m"
```

<a name="cachetypes"></a>
## Cache Types

### Internal

The internal cache uses a builtin type and caches AMIs within the process. If
**AMIQUERY_CACHE_MANAGER** is undefined, the internal cache is used. You can
explicitly define it by setting **AMIQUERY_CACHE_MANAGER** to `internal`.

### Memcached

[memcached](http://memcached.org/) is used to cache AMIs. You must set
 **AMIQUERY_CACHE_MANAGER** to `memcached` and provide the list of
 **AMIQUERY_MEMCACHED_SERVERS** to use memcached.

## RESTful API

`ami-query` leverages vendor mime types to return the RESTful API version to use.

    application/vnd.ami-query-v1+json

See http://blog.pivotal.io/pivotal-labs/labs/api-versioning for more information.

Queries can search by region, ami, status, and tag using the following schema.

    /amis?region=us-west-1&ami=ami-1a2b3c4d&status=available&tag=key:value

If no mime type is specified then `ami-query` will default to the latest API
version.

`status` is also a tag on the AMI, it's provided as a query parameter for
convenience.

You may also specify the `callback` query parameter to receive the output in
JSONP. Additionally, you can specify the `pretty` query parameter to see the
results in a more human friendly format. Note that `callback` and `pretty` are
mutually exclusive with `callback` taking precedence if both are specified.

### Examples

Get all AMIs from all supported regions:

    /amis

Get all AMIs from the `us-west-1` region:

    /amis?region=us-west-1

Get all AMIs from regions `us-west-1` and `us-west-2`:

    /amis?region=us-west-1&region=us-west-2

Get AMI `ami-12345678` from region `us-west-1`:

    /amis?ami=ami-12345678&region=us-west-1

Get all AMIs from region `us-west-1` that have the `status` tag set to
`available`:

    /amis?region=us-west-1&status=available

Get all AMIs from region `us-west-2` that have the tag `yourTag` set to
`yourValue` and the tag `status` set to `development`:

    /amis?region=us-west-2&tag=yourTag:yourValue&status=development

Get all AMIs from region `us-east-1` with the JSONP callback function named
`myCallbackFunc`:

    /amis?region=us-east-1&callback=myCallbackFunc

Get all AMIs from region `us-west-1` and display the results in a more human
readable format:

    /amis?region=us-west-1&pretty
