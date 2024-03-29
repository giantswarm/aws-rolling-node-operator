# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.0] - 2024-03-26

### Added
- Add `global.podSecurityStandards.enforced` value for PSS migration.
### Changed

- Configure `gsoci.azurecr.io` as the default container image registry.
- Upgraded `github.com/prometheus/client_golang` to `v1.11.1`.
- Upgraded `golang.org/x/text` to `v0.3.8`.

## [0.5.0] - 2023-07-13

### Fixed

- Add required values for pss policies.

## [0.4.1] - 2023-07-04

### Added

- Add Service Monitor.

## [0.4.0] - 2023-05-10

### Fixed

- Skip ASG in case no instance is present.

### Changed

- add the use of runtime/default secccomp profile.
- Remove CAPI dependencies.

## [0.3.2] - 2022-12-02

### Fixed 

- Cleanup all annotations.

## [0.3.1] - 2022-12-02

### Fixed

- Bump k8smetadata.

## [0.3.0] - 2022-11-30

### Added

- Support Instance Warmup via Annotation.

## [0.2.0] - 2022-09-20

### Changed

- Use `k8smetadata` annotations.

## [0.1.2] - 2022-09-16

### Fixed

- RBAC permissions.

## [0.1.1] - 2022-09-15

### Fixed

- RBAC permissions.

## [0.1.0] - 2022-09-15

### Added

- Init
- Docs

[Unreleased]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/giantswarm/aws-rolling-node-operator/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/giantswarm/aws-rolling-node-operator/releases/tag/v0.1.0
