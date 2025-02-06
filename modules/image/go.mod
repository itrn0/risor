module github.com/itrn0/risor/modules/image

go 1.22.0

toolchain go1.23.1

replace github.com/itrn0/risor => ../..

require (
	github.com/anthonynsimon/bild v0.14.0
	github.com/itrn0/risor v1.7.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/image v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
