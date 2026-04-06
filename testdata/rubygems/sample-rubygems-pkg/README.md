# sample-rubygems-pkg

A minimal RubyGems package used for **gh-pkg-kit integration testing**.

## Build

```sh
gem build sample-rubygems-pkg.gemspec
```

## Publish

```sh
gem push --key github \
  --host https://rubygems.pkg.github.com/${GITHUB_OWNER} \
  sample-rubygems-pkg-1.0.0.gem
```
