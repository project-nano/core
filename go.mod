module github.com/project-nano/core

go 1.13

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
	github.com/project-nano/framework v1.0.3
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
)
