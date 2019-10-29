## [2.5.0] - 2019-10-28

### Changed

* Updated to golang `1.13.3`.
* Update aws-sdk-go to `v1.25.14`.

### Removed

* Using dep for dependencies (in favor of modules).
* RHEL 6 rpm support.

## [2.4.0] - 2019-03-20

### Added

* Added compression support to API responses.

## [2.3.0] - 2018-08-15

### Added

* Added a config item to disable collecting launch permission information
  (AMIQUERY_COLLECT_LAUNCH_PERMISSIONS).


## [2.2.0] - 2018-04-16

### Changed

* The `tag` query parameter now properly handles tag values with one or more
  colons, e.g. `tag=key:value:of:tag` translates to `key=value:of:tag`.

### Added

* The tag representing an AMI's state is now configurable through the
  `AMIQUERY_STATE_TAG` environment variable. If not provided, the default value
  is `state`.

## [2.1.0] - 2018-03-23

### Added

* Now supports CORS Origin header through the `AMIQUERY_CORS_ALLOWED_ORIGINS`
  environment variable.

## [2.0.1] - 2018-02-13

### Added

* Ability to filter AMIs cached from accounts based on a key-tag. Set via the
  `AMIQUERY_TAG_FILTER` environment variable.

## [2.0.0] - 2017-12-19

### Added

* Ability cache AMIs from mulitple accounts (requires an IAM role).
* Caches the Launch Permissions for an AMI.
* New query paramter, `account_id=123456789012`, to filter on AMIs said account
  has access to.

### Changed

* Now uses [dep][dep] to manage dependencies.

### Removed

* Memcached is no longer an option for caching AMIs. It's strictly an internal
  memory cache now.

## [1.1.0] - 2017-05-23

### Added

* HTTPS support.
* Timeouts to the http.Client to prevent AWS API calls from blocking.

## [1.0.0] - 2015-09-29

* Initial Release.

<!-- links -->
[dep]:https://github.com/golang/dep
