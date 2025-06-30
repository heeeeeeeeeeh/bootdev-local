# bootdev-local

`bootdev-local` is a command-line tool written in Go that allows you to access Boot.dev premium courses and quizzes locally, making them available for free. It provides a convenient way to interact with lesson content and practice coding challenges directly from your terminal.

## Features

  * **Free Access to Premium Lessons:** Access any Boot.dev premium lesson by providing its UUID or direct URL.
  * **Interactive Quizzes:** Engage with quizzes using a `bubbletea` selection interface for an interactive experience.
  * **Local Coding Environment Setup:** For coding lessons, `bootdev-local` automatically creates a structured folder within your current directory (e.g., `./Chapter 6/Lesson 5/main.c`). This folder includes all lesson files, including a `README.md`.
  * **Customizable Editor Integration:** Open code and markdown files directly in your preferred editors using command-line flags.

## Installation

### Arch Linux

If you're on Arch Linux, you can easily install `bootdev-local` using `paru`:

```bash
paru -Sy bootdev-local
```

### Other Linux Distributions

For other Linux distributions, you can download the latest executable directly from the GitHub releases page and place it in your system's `bin` folder:

1.  Download the executable:
    ```bash
    wget https://github.com/Wraient/bootdev-local/releases/download/latest/main -O bootdev-local
    ```
2.  Make it executable:
    ```bash
    chmod +x bootdev-local
    ```
3.  Move it to your `bin` directory (e.g., `/usr/local/bin`):
    ```bash
    sudo mv bootdev-local /usr/local/bin/
    ```

## Usage

### Opening a Lesson

To open a lesson, simply run `bootdev-local` followed by the lesson's UUID or its full URL:

```bash
bootdev-local "bb1b1b68-a688-4341-821c-54614ed5eed2"
# Or
bootdev-local "https://www.boot.dev/lessons/bb1b1b68-a688-4341-821c-54614ed5eed2"
```

### Using Editors

You can specify which editor to use for code files and markdown files using the `-code-editor` and `-md-editor` flags, respectively.

```bash
bootdev-local -h
```

**Output:**

```
Usage of bootdev-local:
  -code-editor string
        Editor to open code files with (e.g., 'code', 'vim', 'emacs')
  -md-editor string
        Editor to open markdown files with (e.g., 'typora', 'code')
```

**Example:**

To open a C coding lesson and have the `main.c` file open in VS Code (`code`) and the `README.md` open in Typora (`typora`):

```bash
bootdev-local "https://www.boot.dev/lessons/your-lesson-uuid" -code-editor "code" -md-editor "typora"
```

This command will:

1.  Fetch the lesson content.
2.  Create a folder structure like `./Chapter 6/Lesson 5/`.
3.  Place `main.c` (and any other relevant files) and `README.md` inside this folder.
4.  Automatically open `main.c` with VS Code and `README.md` with Typora.

-----

## Disclaimer for Boot.dev

This repository is intended solely for individuals who genuinely cannot afford the otherwise very reasonably priced Boot.dev courses. It is not meant to circumvent or undermine the value provided by Boot.dev.

Please understand that the experience of using `bootdev-local` will inherently be degraded compared to the official Boot.dev website, as it lacks the rich interactive features, integrated development environment, and community support that the official platform offers.

If Boot.dev owners have any concerns or would prefer this repository to be private, please feel free to message me on Discord at `@wraient`. I am open to discussion and will respect any reasonable requests.
