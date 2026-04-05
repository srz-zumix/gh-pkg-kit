Gem::Specification.new do |spec|
  spec.name        = "sample-rubygems-pkg"
  spec.version     = "1.0.0"
  spec.authors     = ["srz-zumix"]
  spec.summary     = "Sample RubyGems package for gh-pkg-kit integration testing."
  spec.description = "A minimal gem used for gh-pkg-kit integration testing."
  spec.license     = "MIT"

  spec.metadata["github_repo"] = "ssh://github.com/srz-zumix/gh-pkg-kit"

  spec.files = ["lib/sample_rubygems_pkg.rb"]
end
