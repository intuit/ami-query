# ami-query

[![Build Status](https://travis-ci.org/intuit/ami-query.svg?branch=master)](https://travis-ci.org/intuit/ami-query)

Provide a RESTful API to query information about Amazon AWS AMIs.

## Prerequisites

`ami-query` is written in Go. You need to have version 1.9 or higher installed.
For Go installation instructions see https://golang.org/doc/install.

## Installation

To install `ami-query` export `GOPATH` then use `go get`.

```shell
$ export GOPATH=$HOME/go
$ go get github.com/intuit/ami-query
```

`ami-query` will be installed into `$GOPATH/bin`. If you'd like to install it
into another directory (e.g. `/usr/local/bin`), then you can export `GOBIN`
prior to running `go get`.

To build an RPM for RHEL 7, which can be used for installation on
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

* **AMIQUERY_ROLE_NAME**

  The name of an IAM role `ami-query` will assume (STS AssumeRole) into. This
  role must exist in the accounts specified in **AMIQUERY_OWNER_IDS**. Given the
  account ID `123456789012` and role name `ami-query` the ARN would be
  `arn:aws:iam::123456789012:role/ami-query`.

  The role must include the following permissions for `ami-query` to cache AMIs:

   * ec2:DescribeImageAttribute
   * ec2:DescribeImages

### Optional Values

* **AMIQUERY_LISTEN_ADDRESS**

  The bind address and port that `ami-query` will listen on. The format for this
  value is `ip_addr:port` or `hostname:port`. The default value is
  `localhost:8080`.

* **AMIQUERY_TAG_FILTER**

  The tag-key name used to filter the results of ec2:DescribeImages. The value
  is irrelevant, only the existence of the tag is required.

* **AMIQUERY_STATE_TAG**

  The tag-key name used to determine the state of an AMI. The default value is
  "state".

* **AMIQUERY_REGIONS**

  A comma-separated list of regions that `ami-query` will scan for AMIs. Use
  "us-east-1" for US East, "us-west-1" for US West 1, etc.

* **AMIQUERY_APP_LOGFILE**

   The file location to send application log messages. Note that `ami-query`
   does not manage this file, it only writes to it. The default is to log to
   STDERR.

* **AMIQUERY_HTTP_LOGFILE**

  The file location to send HTTP log messages. Note that `ami-query` does not
  manage this file, it only writes to it. The default is to log to STDERR.

* **AMIQUERY_CORS_ALLOWED_ORIGINS**

  A comma-separated list of allowed Origins.

* **AMIQUERY_COLLECT_LAUNCH_PERMISSIONS**

  If launch permission information should collected for each AMI. The default is
  "true". If you do not want to collect the launch permission information, set
  this to "false".

* **SSL_CERTIFICATE_FILE**

  The file location of the SSL certificate file. **SSL_KEY_FILE** also needs to
  be specified in order to enable HTTPS support.

* **SSL_KEY_FILE**

  The file location of the SSL key file. **SSL_CERTIFICATE_FILE** also needs to
  be specified in order to enable HTTPS support.

#### Cache tunables

The following settings are used to tune the AWS API requests. In accounts with
a large number of AMIs in a single region (> ~150), it's possible there will
sometimes be `RequestLimitExceed` errors when trying to fetch launch
permissions. See the following for more information:

http://docs.aws.amazon.com/AWSEC2/latest/APIReference/query-api-troubleshooting.html#api-request-rate

* **AMIQUERY_CACHE_TTL**

  The time to wait before the cache is updated. The format of this value is a
  duration such as "5s" or "5m". The minimum allowed value is "5m", or 5
  minutes. The default value is "15m".

* **AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS**

  The maximum allowed number of concurrent API requests in a given region for a
  given account owner.

* **AMIQUERY_CACHE_MAX_REQUEST_RETRIES**

  The maximum allowed number of API request retries in a given region for a
  given account owner.

#### Example Configuration

This most basic configuration listens on `localhost:8080`, sets the cache TTL
to 15 minutes, and caches AMIs from all AWS Standard regions. It will attempt to
use AWS credentials from either environment variables, `~/.aws/credentials`, or
the meta-data on an EC2 instance.

```shell
export AMIQUERY_OWNER_IDS="123456789012"
export AMIQUERY_ROLE_NAME="ami-query"
```

This configuration listens on https://localhost:8443, caches AMIs from multiple
accounts, only from two regions, and limits the number of concurrent API
requests to three.

```shell
export AMIQUERY_LISTEN_ADDRESS="localhost:8443"
export AMIQUERY_OWNER_IDS="123456789012,123456789013"
export AMIQUERY_ROLE_NAME="ami-query"
export AMIQUERY_REGIONS="us-east-1,us-west-1"
export AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS="3"
export SSL_CERTIFICATE_FILE="/path/to/tls/certs/ami-query.crt"
export SSL_KEY_FILE="/path/to/tls/private/ami-query.key"
```

## RESTful API

`ami-query` leverages vendor mime types to return the RESTful API version to use.

    application/vnd.ami-query-v1+json

See http://blog.pivotal.io/pivotal-labs/labs/api-versioning for more information.
If no mime type is specified then `ami-query` will default to the latest API
version.

Queries can search by owner_id, region, ami, status, launch_permission, and tag
using the following schema:

    /amis?owner_id=123456789012region=us-west-1&ami=ami-1a2b3c4d&status=available&launch_permission=123456789013&tag=key:value

`status` is also a tag on the AMI, it's provided as a query parameter for
convenience.

`launch_permission` is used to return only the AMIs with the matching launch
permission. If more than one value is provided, only the first value will be
used. If `AMIQUERY_COLLECT_LAUNCH_PERMISSIONS` is "false", this API
functionality will be ignored.

You may also specify the `callback` query parameter to receive the output in
JSONP. Additionally, you can specify the `pretty` query parameter to see the
results in a more human friendly format. Note that `callback` and `pretty` are
mutually exclusive with `callback` taking precedence if both are specified.

### Examples

Get all AMIs from all supported regions:

    /amis

Get all AMIs created by Account ID `123456789012`:

    /amis?owner_id=123456789012

Get AMI `ami-12345678`:

    /amis?ami=ami-12345678

Get all AMIs from the `us-west-1` region:

    /amis?region=us-west-1

Get all AMIs from regions `us-west-1` and `us-west-2`:

    /amis?region=us-west-1&region=us-west-2

Get all AMIs from region `us-west-1` that have the `status` tag set to
`available`:

    /amis?region=us-west-1&status=available

Get all AMIs from region `us-west-2` that Account ID `123456789012` has
permission to launch:

    /amis?region=us-west-2&launch_permission=123456789012

Get all AMIs from region `us-west-2` that have the tag `yourTag` set to
`yourValue` and the tag `status` set to `development`:

    /amis?region=us-west-2&tag=yourTag:yourValue&status=development

Get all AMIs from region `us-east-1` with a JSONP callback function named
`myCallbackFunc`:

    /amis?region=us-east-1&callback=myCallbackFunc

Get all AMIs from region `us-west-1` and display the results in a more human
readable format:

    /amis?region=us-west-1&pretty

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
