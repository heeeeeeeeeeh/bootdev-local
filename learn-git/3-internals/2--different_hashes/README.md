# Different Hashes

My latest Git commit hash was:

```bash
5ba786fcc93e8092831c01e71444b9baa2228a4f
```

You may have noticed that even though we (you and I) both have the **same content** in our repositories, we have **different commit hashes**. While commit hashes _are_ derived from their content changes, there's [also some other stuff](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects#_git_commit_objects) that affects the end hash. For example:

- The commit message
- The author's name and email
- The date and time
- Parent (previous) commit hashes

All this to say that hashes are (almost) always unique, and because they're generated automatically for you, you don't need to worry too much about what goes into them right now.

## Note: Hash = SHA

Git uses a cryptographic hash function called [SHA-1](https://en.wikipedia.org/wiki/SHA-1) to generate commit hashes. We won't go into the details of how SHA-1 works in this course, but it's important to know because you might also hear commit hashes referred to as "SHAs".
