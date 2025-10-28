module github.com/pmatteo/chiserver_example

go 1.25

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/pmatteo/chiserver v0.0.0
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/pmatteo/chiserver => ../
