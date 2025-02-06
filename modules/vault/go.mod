module github.com/itrn0/risor/modules/vault

go 1.22.0

toolchain go1.23.1

replace github.com/itrn0/risor => ../..

require (
	github.com/hashicorp/vault-client-go v0.4.3
	github.com/itrn0/risor v1.7.0
)

require (
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/time v0.6.0 // indirect
)
