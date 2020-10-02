module github.com/project-nano/core

go 1.13

replace (
	github.com/project-nano/core/imageserver => ./src/imageserver
	github.com/project-nano/core/modules => ./src/modules
	github.com/project-nano/core/task => ./src/task
	github.com/project-nano/framework => ../framework
)

require (
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/reedsolomon v1.9.3 // indirect
	github.com/project-nano/core/imageserver v0.0.0-00010101000000-000000000000
	github.com/project-nano/core/modules v0.0.0-00010101000000-000000000000
	github.com/project-nano/core/task v0.0.0-00010101000000-000000000000
	github.com/project-nano/framework v1.0.3
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sevlyar/go-daemon v0.1.5 // indirect
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
)
