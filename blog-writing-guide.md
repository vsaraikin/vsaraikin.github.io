# Technical Blog Writing Guide

A collection of best practices and patterns extracted from widely-recommended technical blogs.

## Recommended Blogs to Study

### Systems Programming & Low-Level
- **[saminiir.com](https://www.saminiir.com)** - TCP/IP stack implementation, kernel development
- **[Eli Bendersky's website](https://eli.thegreenplace.net)** - Compilers, Go internals, Python
- **[Dan Luu](https://danluu.com)** - Systems, performance, industry analysis
- **[Brendan Gregg](https://www.brendangregg.com)** - Performance engineering, eBPF, tracing
- **[Phil Eaton](https://notes.eatonphil.com)** - Database internals, interpreters

### Web Development & JavaScript
- **[Overreacted (Dan Abramov)](https://overreacted.io)** - React, mental models, career
- **[Kent C. Dodds](https://kentcdodds.com)** - Testing, React patterns
- **[Josh Comeau](https://www.joshwcomeau.com)** - CSS, animations, interactive tutorials

### Explanatory & Educational
- **[Julia Evans](https://jvns.ca)** - Linux, networking, zines
- **[Wizard Zines](https://wizardzines.com)** - Visual explanations of technical concepts
- **[Wait But Why](https://waitbutwhy.com)** - Long-form explanations with stick figures

### Company Engineering Blogs
- **[Netflix Tech Blog](https://netflixtechblog.com)** - Scale, microservices, streaming
- **[Cloudflare Blog](https://blog.cloudflare.com)** - Networking, security, performance
- **[Fly.io Blog](https://fly.io/blog)** - Infrastructure, distributed systems

---

## Writing Principles

### 1. Start with "Why Should I Care?"

Don't start with definitions. Start with the problem.

**Bad:**
```
Redis is an in-memory data structure store that supports
strings, hashes, lists, sets, and sorted sets.
```

**Good:**
```
Your API is slow. Every request hits the database.
You need caching. Here's how Redis solves this.
```

### 2. One Idea Per Post

The best technical posts explain ONE thing well. If you're covering multiple concepts, split them into a series.

- "What is a goroutine?" - one post
- "How does the Go scheduler work?" - another post
- "Debugging goroutine leaks" - another post

### 3. Write for Your Past Self

Target the person you were 6 months ago. This ensures:
- You know exactly what they don't know
- You remember what was confusing
- You can anticipate their questions

### 4. Show, Don't Tell

**Bad:**
```
False sharing occurs when threads modify variables
that share the same cache line.
```

**Good:**
```
False sharing occurs when threads modify variables
that share the same cache line.

var counter1 int64  // Thread 1 writes here
var counter2 int64  // Thread 2 writes here
// Both are on the same 64-byte cache line!

Benchmark:
Shared cache line:    2,847 ns/op
Separate cache lines:   89 ns/op  (32x faster)
```

### 5. Use Diagrams Liberally

ASCII art is fine. Perfect diagrams aren't necessary.

```
┌─────────────┐     ┌─────────────┐
│   Client    │────▶│   Server    │
└─────────────┘     └─────────────┘
       │                   │
       │   TCP Handshake   │
       │◀─────────────────▶│
```

Tools:
- [Excalidraw](https://excalidraw.com) - hand-drawn style
- [ASCII Flow](https://asciiflow.com) - ASCII diagrams
- [Mermaid](https://mermaid.js.org) - code-based diagrams

### 6. Code Snippets Must Be Runnable

Every code example should:
- Compile/run without modifications
- Have syntax highlighting
- Be minimal (remove unnecessary parts)
- Include output when relevant

**Bad:**
```go
// ... some code ...
result := doSomething(data)
// ... more code ...
```

**Good:**
```go
package main

func main() {
    data := []int{1, 2, 3}
    result := sum(data)
    fmt.Println(result) // Output: 6
}

func sum(nums []int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}
```

### 7. Benchmark Claims

Never say "X is faster than Y" without numbers.

```
BenchmarkMap-8         5000000    312 ns/op
BenchmarkSyncMap-8     3000000    489 ns/op

map is 1.5x faster for single-threaded access.
```

### 8. Admit What You Don't Know

Readers trust writers who acknowledge limitations.

> "I haven't tested this on ARM. The cache line size
> might be different."

> "This approach works for our scale (10K req/s).
> At 1M req/s, you'd need a different architecture."

---

## Structure Patterns

### The "What Every Programmer Should Know" Pattern

Used by: Ulrich Drepper's memory paper, many viral posts

1. Start with fundamental concept everyone uses
2. Reveal the hidden complexity underneath
3. Show practical implications
4. Provide actionable takeaways

### The Julia Evans Pattern

1. Title is a question: "What happens when you run a program?"
2. Short paragraphs (2-3 sentences max)
3. Lots of bullet points
4. Hand-drawn or simple diagrams
5. "I was confused about X, here's what I learned"
6. End with "things I still don't understand"

### The Dan Abramov Pattern

1. Personal narrative ("I used to think...")
2. Concrete example that breaks intuition
3. Build mental model step by step
4. Connect back to real-world implications
5. Short posts (5-10 minute read)

### The Deep Dive Pattern

Used by: Eli Bendersky, saminiir

1. "Let's build X from scratch"
2. Start with simplest possible version
3. Add complexity incrementally
4. Show actual code at each step
5. End with production-ready version

---

## Formatting Guidelines

### Length
- **Short posts**: 5-10 minutes (800-1500 words)
- **Deep dives**: 20-30 minutes (3000-5000 words)
- **Series**: Split into 5-10 minute chunks

### Headlines
- Use questions: "Why is my Docker build slow?"
- Be specific: "Reducing Go binary size from 100MB to 10MB"
- Avoid clickbait: Not "You won't believe this Go trick"

### Code Blocks
- Always specify language for syntax highlighting
- Keep under 30 lines when possible
- Add comments for non-obvious parts
- Show output/results

### Paragraphs
- Max 3-4 sentences per paragraph
- One idea per paragraph
- Use line breaks liberally

### Lists
- Use bullet points for unordered items
- Use numbered lists for sequential steps
- Don't nest more than 2 levels deep

---

## Things to Avoid

### 1. Wall of Text
Break up long explanations with:
- Code examples
- Diagrams
- Bullet points
- Section headers

### 2. Assuming Knowledge
Define acronyms on first use:
> "TLB (Translation Lookaside Buffer) is a cache for page table entries."

### 3. Over-Explaining Basics
Link to resources instead:
> "If you're new to goroutines, see [Effective Go](https://go.dev/doc/effective_go)."

### 4. No Practical Examples
Every concept needs a "when would I use this?" example.

### 5. Outdated Information
- Include date in post metadata
- Note Go/language version for code
- Update or mark posts as outdated

---

## Pre-Publish Checklist

- [ ] Does the title clearly describe the content?
- [ ] Is there a hook in the first paragraph?
- [ ] Can all code examples be copy-pasted and run?
- [ ] Are diagrams clear without zooming?
- [ ] Did you proofread for typos?
- [ ] Is the post useful to someone who lands on it from Google?
- [ ] Would you share this post if someone else wrote it?

---

## Writing Process

1. **Draft quickly** - Get ideas out, don't edit
2. **Add code examples** - Make abstract concepts concrete
3. **Add diagrams** - Visual learners exist
4. **Cut ruthlessly** - Remove anything not essential
5. **Read aloud** - Catches awkward phrasing
6. **Sleep on it** - Fresh eyes find problems
7. **Get feedback** - One technical reader, one non-expert

---

## Resources

### Tools
- **Grammarly** - Catch grammar issues
- **Hemingway App** - Simplify complex sentences
- **Carbon** - Beautiful code screenshots
- **Excalidraw** - Quick diagrams

### Books
- "On Writing Well" by William Zinsser
- "The Elements of Style" by Strunk & White
- "Technical Writing" by Gerald Alred

### Posts About Writing
- [Write Like You Talk](http://www.paulgraham.com/talk.html) - Paul Graham
- [Writing Well](https://www.julian.com/guide/write/intro) - Julian Shapiro
- [The Day You Became a Better Writer](https://dilbertblog.typepad.com/the_dilbert_blog/2007/06/the_day_you_bec.html) - Scott Adams
