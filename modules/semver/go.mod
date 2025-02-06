module github.com/itrn0/risor/modules/semver

go 1.22.0

toolchain go1.23.1

replace github.com/itrn0/risor => ../..

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/itrn0/risor v1.7.0
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
