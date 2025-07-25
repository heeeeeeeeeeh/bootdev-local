# Empty Struct

[Empty structs](https://dave.cheney.net/2014/03/25/the-empty-struct) are used in Go as a [unary](https://en.wikipedia.org/wiki/Unary_operation) value.

```go

// anonymous empty struct type
empty := struct{}{}

// named empty struct type
type emptyStruct struct{}
empty := emptyStruct{}
```

The cool thing about empty structs is that they're the smallest possible type in Go: they take up **zero bytes of memory**.

![memory usage](https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/hXAvfvS-1280x448.png)

Later in this course, you'll see how and when they're used: it's surprisingly often! Mostly with maps and channels.
