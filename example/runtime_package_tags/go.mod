module github.com/widmogrod/mkunion/example/runtime_package_tags

go 1.23.0

require github.com/widmogrod/mkunion v0.0.0

require (
	github.com/sashabaranov/go-openai v1.40.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/widmogrod/mkunion => ../../
