## [Unreleased]

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