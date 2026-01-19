SHELL := /bin/bash

generate: tools/templ/bin/templ
	tools/templ/bin/templ generate

bin/templ: tools/templ/bin/templ
	mkdir -p bin
	ln -fs $(PWD)/tools/templ/bin/templ bin/templ

tools/templ/bin/templ:
	mkdir -p tools/templ/bin
	GOBIN=$(PWD)/tools/templ/bin go install github.com/a-h/templ/cmd/templ@v0.3.960

