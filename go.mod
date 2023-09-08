module github.com/project-nano/core

go 1.19

replace (
	github.com/project-nano/core/imageserver => ./src/imageserver
	github.com/project-nano/core/modules => ./src/modules
	github.com/project-nano/core/task => ./src/task
	github.com/project-nano/framework => ../framework
)

require (
	github.com/project-nano/core/imageserver v0.0.0-00010101000000-000000000000
	github.com/project-nano/core/modules v0.0.0-00010101000000-000000000000
	github.com/project-nano/core/task v0.0.0-00010101000000-000000000000
	github.com/project-nano/framework v1.0.9
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
)

require (
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/klauspost/reedsolomon v1.11.8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sevlyar/go-daemon v0.1.6 // indirect
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/xor v0.0.0-20191217153810-f85b25db303b // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/xtaci/kcp-go v5.4.20+incompatible // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
