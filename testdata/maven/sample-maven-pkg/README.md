# sample-maven-pkg

A minimal Maven package used for **gh-pkg-kit integration testing**.

## Build

```sh
mvn package
```

## Publish

```sh
GITHUB_ACTOR=<username> GITHUB_TOKEN=<token> \
  mvn deploy \
    -s settings.xml \
    -DGITHUB_OWNER=<owner> \
    -DGITHUB_REPO=<repo>
```
