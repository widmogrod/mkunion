---
title: Contributing and development
---

# Contributing and development
## Development

Checkout repo and run:
```
./dev/bootstrap.sh
```

This command starts docker container with all necessary tools to develop and test `mkunion` project.

In separate terminal run:
```
go generate ./...
go test ./...
```

This will generate code and run tests.

Note: Some tests are flaky (yes I know, I'm working on it), so if you see some test failing, please run it again.

## Documentation

To preview documentation run:
```
 ./dev/docs.sh run
```

## Contributing

If you want to contribute to `mkunion` project, please open issue first to discuss your idea.
I have opinions about how `mkunion` should work, how I want to evolve it, and I want to make sure that your idea fits into the project.
