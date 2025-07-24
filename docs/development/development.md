---
title: Contributing and development
---

# Contributing and development
## Contributing

If you want to contribute to the `mkunion` project, please open an issue first to discuss your idea.

I have opinions about how `mkunion` should work, how I want to evolve it, and I want to make sure that your idea fits into the project.

## Development

Checkout the repo and run:
```
./dev/bootstrap.sh
```

This command starts a Docker container with all the necessary tools to develop and test the `mkunion` project.

In a separate terminal, run:
```
echo "Build mkunion ..."
go build -C cmd/mkunion .

echo "Generate files ..."
cmd/mkunion/mkunion watch -g ./...

echo "Add vaiables manualy..."
source .envrc
# or use:
#  direnv allow .envrc

echo "Run tests ..."
go test ./...
```

This will generate code and run tests.

Note: Some tests may be flaky (this is a known issue being addressed). If you encounter a failing test, please try running it again.

## Documentation

To preview the documentation, run:
```
./dev/docs.sh run
```
