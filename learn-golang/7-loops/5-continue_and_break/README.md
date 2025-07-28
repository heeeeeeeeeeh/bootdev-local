# Continue & Break

Whenever we want to change the control flow of a loop we can use the `continue` and `break` keywords.

## continue

The `continue` keyword stops the current iteration of a loop and continues to the next iteration. `continue` is a powerful way to use the [guard clause](https://blog.boot.dev/clean-code/guard-clauses/) pattern within loops.

```go
for i := 0; i < 10; i++ {
  if i % 2 == 0 {
    continue
  }
  fmt.Println(i)
}
// 1
// 3
// 5
// 7
// 9
```

## break

The `break` keyword stops the current iteration of a loop and exits the loop.

```go
for i := 0; i < 10; i++ {
  if i == 5 {
    break
  }
  fmt.Println(i)
}
// 0
// 1
// 2
// 3
// 4
```

## Assignment

As an Easter egg, we decided to reward our users with a free text message if they send a [prime number](https://en.wikipedia.org/wiki/Prime_number) of text messages this year.

Complete the `printPrimes` function. It should print all of the prime numbers up to and including `max`. It should skip any numbers that are not prime.

Here's the pseudocode:

```
printPrimes(max):
  for n in range(2, max+1):
    if n is 2:
      n is prime, print it
    if n is even:
      n is not prime, skip to next n
    for i in range (3, sqrt(n), 2):
      if n can be evenly divided by i:
        n is not prime, skip to next n
    n is prime, print it
```

### [Breakdown](https://en.wikipedia.org/wiki/Primality_test)

- We skip even numbers because they can't be prime
- We only check up to the square root of `n`. A factor _higher_ than the square root of `n` must multiply with a factor _lower_ than the square root of `n`, meaning we only need to check up to the square root of `n` for potential factors.
  - In your code, you can set the stop condition as `i * i <= n`
- We start checking at 2 because 1 is not prime

> [!warning]
> This lesson is graded based on the output of your program, so don't leave any debugging print statements in your code.

### Example Output

For the first test case, prime number up to 10:

```
Primes up to 10:
2
3
5
7
===============================================================
```

We only want you to print the numbers themselves, not the headings and delimiter, they are already handled for you in the test code.

## Tip

- For the inner loop when a number has a factor, you should use the `break` keyword to exit the loop early.
