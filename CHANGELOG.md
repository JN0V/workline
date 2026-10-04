# Changelog

## [0.5.0](https://github.com/JN0V/workline/compare/v0.4.0...v0.5.0) (2026-10-04)


### Features

* **documentalist:** list the docs waiting for a person in one issue ([dde4264](https://github.com/JN0V/workline/commit/dde42647ea7ce8b1878bfba09406308c6cd61435))
* **engine:** close an issue, its evidence quoted, by mode and cap ([b92c3a4](https://github.com/JN0V/workline/commit/b92c3a412fc6de65808afa6f0b217a4ce592b257))
* **forge:** list open issues, read comments, close with a reason ([598490f](https://github.com/JN0V/workline/commit/598490f593fcf63559c1a3b7305891096fdec02c))
* **forge:** list open milestones, put an issue in one ([25f5be5](https://github.com/JN0V/workline/commit/25f5be59124e78f02ba5e706b986b08daaa11575))
* **product-owner:** give the agent the code an issue names ([571af87](https://github.com/JN0V/workline/commit/571af875779d41d950069b16a75adcb7e245d9ce))
* **product-owner:** import a backlog file by reading it, not parsing it ([b1f9880](https://github.com/JN0V/workline/commit/b1f98804a7c86ce7bbe38616e0659ad200e46ac2))
* **product-owner:** name the code an issue is about, quoted ([11d5457](https://github.com/JN0V/workline/commit/11d54575bd9a2e4eeeef9cb471c06d111a3dd73c))
* **product-owner:** put an issue in a release's milestone ([9026e91](https://github.com/JN0V/workline/commit/9026e91576ceb59314b0a993f0ebc5cb7699fd85))
* **product-owner:** read a share of the backlog a run, again on change ([cfa827b](https://github.com/JN0V/workline/commit/cfa827b6e192bdc0db1d4be7d877a2f030dcd330))
* **product-owner:** the role, closing duplicates and proposing obsolete ([84abfe6](https://github.com/JN0V/workline/commit/84abfe6e2495c0ce02cba2af5f34ec0279924bfd))


### Bug Fixes

* **documentalist:** a release tool's version bump makes no doc suspect ([545b60a](https://github.com/JN0V/workline/commit/545b60ae74ce5883dcbd9804abffa1d4a402d42d))
* **product-owner:** an unreadable answer reads no issue ([9510691](https://github.com/JN0V/workline/commit/95106912335d6b7225364571e363624f355f00f5))
* **product-owner:** give duplicate's original whole, though read before ([b2863af](https://github.com/JN0V/workline/commit/b2863afe321cd5257d7ffba1933be675cc62152e))
* **product-owner:** keep a proposal in the report until it is settled ([104e79e](https://github.com/JN0V/workline/commit/104e79e1e624c63383a22fbb7e00db2766bc490b))
* **product-owner:** see an undone closing and a new comment at once ([a03d3d4](https://github.com/JN0V/workline/commit/a03d3d4f86959d7cd8ffa88273d10c9a26fc36fd))

## [0.4.0](https://github.com/JN0V/workline/compare/v0.3.0...v0.4.0) (2026-10-03)


### Features

* **documentalist:** hold from the highest release tag, not the nearest ([4dd4744](https://github.com/JN0V/workline/commit/4dd474490bd1a9e97ce90aa3e02dd6fd78acc7e7))
* **engine:** fix a release pull request in a merge request of its own ([c79e70d](https://github.com/JN0V/workline/commit/c79e70d83b2a449c1c67aaa257078aa642ce8aab))
* **engine:** hold a release tool's pull request as the release ([7b02d13](https://github.com/JN0V/workline/commit/7b02d13d4f4dccb57fea52d6424a8a257a39e957))
* **forge:** say which branch a merge request goes into ([f9c46bb](https://github.com/JN0V/workline/commit/f9c46bb13ef53bb2329b2784dfa84b7affd2fed8))
* hold a release tool's pull request as the release (ADR-0017 step 4) ([c225a9f](https://github.com/JN0V/workline/commit/c225a9f8287fa3469cc2d93f9adb28e121f3e43e))


### Bug Fixes

* **documentalist:** say release event is for the release, not gardening ([e2f681d](https://github.com/JN0V/workline/commit/e2f681da8fcded40c0ec15eb6cf50d44adbe35b2))
* **engine:** never ask again for a block the answer did not cause ([a7ba9e5](https://github.com/JN0V/workline/commit/a7ba9e516a843f98ba4068bec553f07e26191adb))
* **engine:** never ask again for a block the answer did not cause ([addaa0c](https://github.com/JN0V/workline/commit/addaa0c484ff43507f3f3a59caf3fb05cccfdacb))

## [0.3.0](https://github.com/JN0V/workline/compare/v0.2.3...v0.3.0) (2026-10-03)


### ⚠ BREAKING CHANGES

* **roles:** retire the release manager for the project's release tool

### Features

* **roles:** retire the release manager for the project's release tool ([8274c85](https://github.com/JN0V/workline/commit/8274c85d035f14028418fb373bf4b9b9d9c7e4b7))


### Bug Fixes

* **documentalist:** hold the release for a release tool, not a role ([aaddf17](https://github.com/JN0V/workline/commit/aaddf17109e51b7b1f4acae9b226e5dd2d619e8e))
