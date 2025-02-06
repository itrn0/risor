module github.com/itrn0/risor/modules/jmespath

go 1.22.0

toolchain go1.23.1

replace github.com/itrn0/risor => ../..

require (
	github.com/jmespath-community/go-jmespath v1.1.1
	github.com/itrn0/risor v1.7.0
)

require golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
