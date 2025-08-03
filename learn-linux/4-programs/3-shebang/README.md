# Shebang

As we talked about before, you can run any executable file by typing its file path into your shell. For example:

```bash
bin/genids.sh
```

That works out-of-the-box for files that are compiled executables. But what about scripts that need to be interpreted by another program? The computer needs to be told what program to use to interpret the file.

A ["shebang"](<https://en.wikipedia.org/wiki/Shebang_(Unix)>) is a special line at the top of a script that tells your shell which program to use to execute the file.

The format of a shebang is:

```bash
#! interpreter [optional-arg]
```

For example, if your script is a Python script and you want to use Python 3, your shebang might look like this:

```bash
#!/usr/bin/python3
```

This tells the system to use the Python 3 interpreter located at `/usr/bin/python3` to run the script.

## Assignment

Use the `cat` command to view the contents of the `private/bin/genids.sh` file.

Paste _only_ the shebang (1 line) into the input field and submit your answer.
