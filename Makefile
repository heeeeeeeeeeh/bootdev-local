CC = gcc
CFLAGS = -Wall -Wextra -I./deps/munit
LDFLAGS = 

# Munit source file
MUNIT_SRC = deps/munit/munit.c

# Find all exercise directories
EXERCISE_DIRS := $(shell find . -type f -name "main.c" -exec dirname {} \;)

.PHONY: all clean setup $(EXERCISE_DIRS)

all: setup $(EXERCISE_DIRS)

# Setup dependencies
setup:
	@if [ ! -d "deps/munit" ]; then \
		echo "Setting up Munit..."; \
		mkdir -p deps; \
		cd deps && \
		git clone https://github.com/nemequ/munit.git || true; \
	fi

# Rule to build each exercise
$(EXERCISE_DIRS):
	@echo "Building $@..."
	@$(CC) $(CFLAGS) $@/main.c $@/exercise.c $(MUNIT_SRC) -o $@/solution $(LDFLAGS)

# Clean all built files
clean:
	@echo "Cleaning..."
	@find . -type f -name "solution" -delete
	@find . -type f -name "*.o" -delete

# Example usage:
# make              - sets up deps and builds all exercises
# make chapter1/ex1 - builds specific exercise
# make clean        - cleans all built files 