#!/bin/bash
set -e

cwd=$(dirname "$0")
project_root=$(dirname "$cwd")

if [ "$1" == "run" ]; then
  docker run --rm -it -p 8000:8000 -v ${project_root}:/docs squidfunk/mkdocs-material
elif [ "$1" == "build" ]; then
  docker run --rm -it -v ${project_root}:/docs squidfunk/mkdocs-material build
else
  echo "Usage: $0 [run|build]"
fi