---
title: "Linux perf: The Swiss Army Knife of Performance Analysis"
date: 2025-01-22
draft: true
description: "A practical guide to Linux perf: CPU profiling, hardware counters, flame graphs, and finding performance bottlenecks."
---

Your program is slow. But where? Which function? Which line?

`perf` answers these questions. It's the standard profiler on Linux, built into the kernel, and more powerful than most developers realize.

This guide covers practical `perf` usage: from basic CPU profiling to hardware counter analysis and flame graph generation.

## What is perf?

`perf` (also called perf_events or PCL - Performance Counters for Linux) is a profiling tool that uses CPU hardware counters and kernel tracepoints to measure program performance.

It can:
- Count CPU cycles, cache misses, branch mispredictions
- Sample which functions are using CPU time
- Trace system calls, context switches, page faults
- Generate flame graphs for visualization

Unlike tools that instrument your code (adding overhead), `perf` uses hardware sampling with minimal impact on performance.

## Installation

```bash
# Ubuntu/Debian
sudo apt install linux-tools-common linux-tools-$(uname -r)

# Fedora/RHEL
sudo dnf install perf

# Arch
sudo pacman -S perf

# Verify installation
perf --version
```

For full functionality, you may need to adjust permissions:

```bash
# Allow non-root users to use perf (temporary)
sudo sysctl -w kernel.perf_event_paranoid=-1

# Or permanently in /etc/sysctl.conf
echo 'kernel.perf_event_paranoid=-1' | sudo tee -a /etc/sysctl.conf
```

## The Basics: perf stat

`perf stat` runs a command and shows CPU statistics when it completes.

```bash
$ perf stat ./my_program

 Performance counter stats for './my_program':

          1,542.31 msec task-clock                #    0.998 CPUs utilized
                42      context-switches          #   27.232 /sec
                 3      cpu-migrations            #    1.946 /sec
            12,847      page-faults               #    8.330 K/sec
     5,124,891,234      cycles                    #    3.324 GHz
     8,847,291,847      instructions              #    1.73  insn per cycle
     1,284,729,184      branches                  #  833.004 M/sec
        12,847,291      branch-misses             #    1.00% of all branches

       1.545892341 seconds time elapsed
       1.532847000 seconds user
       0.012000000 seconds sys
```

Key metrics:
- **Instructions per cycle (IPC)**: Higher is better. >2 is good, <1 indicates stalls
- **Branch misses**: >5% suggests unpredictable control flow
- **Context switches**: High numbers may indicate I/O blocking

### Specific Events

```bash
# Cache analysis
perf stat -e cache-references,cache-misses,L1-dcache-loads,L1-dcache-load-misses ./program

# Branch prediction
perf stat -e branches,branch-misses ./program

# Memory
perf stat -e dTLB-loads,dTLB-load-misses,page-faults ./program

# Everything useful
perf stat -e cycles,instructions,cache-references,cache-misses,branches,branch-misses ./program
```

### Example: Detecting Cache Problems

```bash
$ perf stat -e cache-references,cache-misses ./random_access

     892,847,291      cache-references
     284,729,184      cache-misses              #   31.89% of all cache refs

$ perf stat -e cache-references,cache-misses ./sequential_access

     892,847,291      cache-references
       8,472,918      cache-misses              #    0.95% of all cache refs
```

A 31% cache miss rate vs 0.95% - that's your bottleneck.

## CPU Profiling: perf record + perf report

`perf record` samples your program's execution. `perf report` shows where time was spent.

```bash
# Record with call graphs (stack traces)
perf record -g ./my_program

# View results
perf report
```

The interactive `perf report` UI:

```
Samples: 42K of event 'cycles', Event count (approx.): 28472918472
Overhead  Command      Shared Object        Symbol
  24.32%  my_program   my_program           [.] process_data
  18.47%  my_program   my_program           [.] hash_lookup
  12.83%  my_program   libc.so.6            [.] malloc
   8.29%  my_program   my_program           [.] parse_input
   7.14%  my_program   libc.so.6            [.] memcpy
```

Press Enter on a function to see its call graph (who calls it, what it calls).

### Recording Options

```bash
# Sample at 99 Hz (default is 4000 Hz, 99 avoids lockstep with timers)
perf record -F 99 -g ./program

# Record system-wide for 10 seconds
perf record -F 99 -a -g -- sleep 10

# Record specific PID
perf record -F 99 -g -p 1234

# With DWARF unwinding (better stack traces for optimized code)
perf record --call-graph dwarf ./program

# Specific CPU cores
perf record -C 0,1 ./program
```

### Understanding the Output

```
-   24.32%    24.32%  my_program  my_program      [.] process_data
   - 24.32% process_data
      - 18.47% called_from_main
         + 12.83% main
      + 5.85% called_from_thread
```

- **Overhead**: Percentage of samples in this function
- **Self**: Time in the function itself (not children)
- **Children**: Time in this function + everything it calls
- `-`: Expandable call tree
- `+`: Collapsed call tree

### Useful perf report Options

```bash
# Sort by self time (not children)
perf report --no-children

# Show source code annotations
perf report --stdio

# Filter by symbol
perf report -s symbol

# Show specific columns
perf report --fields=overhead,symbol
```

## Real-Time Profiling: perf top

Like `top`, but for functions:

```bash
# System-wide
sudo perf top

# Specific process
perf top -p $(pgrep my_program)

# With call graphs
perf top -g
```

Output:

```
Samples: 82K of event 'cycles', 4000 Hz, Event count: 41847291847
Overhead  Shared Object        Symbol
  12.34%  [kernel]             [k] _raw_spin_lock
   8.47%  my_program           [.] hot_function
   6.29%  libc.so.6            [.] __memmove_avx
   4.18%  [kernel]             [k] copy_user_generic
```

## Flame Graphs

Flame graphs are the best way to visualize profiling data. They show:
- X-axis: Proportion of CPU time (wider = more time)
- Y-axis: Call stack depth (bottom = entry point)

### Generating Flame Graphs

```bash
# 1. Record with stack traces
perf record -F 99 -g ./my_program

# 2. Convert to text
perf script > out.perf

# 3. Generate flame graph (need FlameGraph tools)
git clone https://github.com/brendangregg/FlameGraph
./FlameGraph/stackcollapse-perf.pl out.perf > out.folded
./FlameGraph/flamegraph.pl out.folded > flamegraph.svg

# Open in browser
firefox flamegraph.svg
```

Or use the built-in (newer kernels):

```bash
perf record -F 99 -g ./my_program
perf script report flamegraph
```

### Reading Flame Graphs

```
┌──────────────────────────────────────────────────────────────────┐
│                          malloc (12%)                            │
├──────────────────────────────────────────────────────────────────┤
│              hash_lookup (18%)           │    parse_json (8%)    │
├──────────────────────────────────────────┼───────────────────────┤
│                    process_data (35%)                            │
├──────────────────────────────────────────────────────────────────┤
│                          main (100%)                             │
└──────────────────────────────────────────────────────────────────┘
```

- **Wide boxes** = lots of CPU time = optimization targets
- **Tall stacks** = deep call chains
- **Plateaus** = time spent in that exact function (not children)

## Hardware Events Deep Dive

`perf` can measure thousands of hardware events. List them:

```bash
perf list

# Hardware events
  cpu-cycles OR cycles
  instructions
  cache-references
  cache-misses
  branch-instructions OR branches
  branch-misses
  bus-cycles
  stalled-cycles-frontend
  stalled-cycles-backend

# Hardware cache events
  L1-dcache-loads
  L1-dcache-load-misses
  L1-icache-load-misses
  LLC-loads
  LLC-load-misses
  dTLB-loads
  dTLB-load-misses
```

### Cache Analysis

```bash
# L1 data cache
perf stat -e L1-dcache-loads,L1-dcache-load-misses ./program

# Last-level cache (L3)
perf stat -e LLC-loads,LLC-load-misses ./program

# TLB misses (memory-mapped workloads)
perf stat -e dTLB-loads,dTLB-load-misses ./program
```

### Sample on Cache Misses

Instead of sampling CPU cycles, sample when cache misses happen:

```bash
# Record where cache misses occur
perf record -e cache-misses -g ./program
perf report
```

This shows which functions cause the most cache misses.

### Branch Prediction

```bash
perf stat -e branches,branch-misses ./program

# Sample on branch misses
perf record -e branch-misses -g ./program
```

## Tracing

`perf` can also trace kernel and user events.

### System Calls

```bash
# Trace all syscalls
perf trace ./program

# Specific syscalls
perf trace -e read,write ./program
```

### Context Switches

```bash
# Count context switches
perf stat -e context-switches ./program

# Trace each context switch
perf record -e sched:sched_switch -a -- sleep 5
```

### Page Faults

```bash
# Count page faults
perf stat -e page-faults,minor-faults,major-faults ./program

# Trace where page faults happen
perf record -e page-faults -g ./program
```

## Practical Examples

### Example 1: Finding a Hot Function

```bash
$ perf record -g ./slow_program
$ perf report --no-children

Overhead  Symbol
  45.23%  slow_function
  12.84%  helper_function
   8.47%  malloc
```

`slow_function` uses 45% of CPU time. Look there first.

### Example 2: Cache Miss Analysis

Compile with debug info:

```bash
g++ -O2 -g program.cpp -o program
```

Profile cache misses:

```bash
$ perf stat -e cache-references,cache-misses ./program

    847,291,847      cache-references
    284,729,184      cache-misses    # 33.61%
```

Find where they happen:

```bash
$ perf record -e cache-misses -g ./program
$ perf report

Overhead  Symbol
  62.34%  random_access_loop    # This is the problem
  18.47%  hash_table_lookup
```

Annotate to see which lines:

```bash
$ perf annotate random_access_loop

       │    for (int i = 0; i < N; i++) {
 62.34 │      sum += arr[indices[i]];  // <-- Cache misses here
       │    }
```

### Example 3: Comparing Two Implementations

```bash
# Version A
$ perf stat -e cycles,instructions,cache-misses ./version_a
     5,847,291,847      cycles
     8,472,918,472      instructions    # 1.45 IPC
       284,729,184      cache-misses

# Version B
$ perf stat -e cycles,instructions,cache-misses ./version_b
     2,847,291,847      cycles
    12,472,918,472      instructions    # 4.38 IPC
         8,472,918      cache-misses
```

Version B: 2x fewer cycles, 3x more IPC, 33x fewer cache misses. Clear winner.

### Example 4: Finding Memory Allocation Overhead

```bash
$ perf record -g ./program
$ perf report

Overhead  Symbol
  28.47%  malloc
  12.34%  free
   8.47%  operator new
```

40%+ in allocation? Consider object pooling or arena allocation.

## One-Liners Reference

```bash
# CPU Statistics
perf stat command                    # Basic CPU stats
perf stat -d command                 # Detailed stats
perf stat -e EVENT1,EVENT2 command   # Specific events

# Profiling
perf record command                  # Sample CPU usage
perf record -g command               # With call graphs
perf record -F 99 -g command         # At 99 Hz
perf record --call-graph dwarf cmd   # DWARF unwinding

# Analysis
perf report                          # Interactive report
perf report --stdio                  # Text report
perf annotate function               # Source annotation

# Real-time
perf top                             # Live CPU profile
perf top -p PID                      # Specific process

# Tracing
perf trace command                   # Trace syscalls
perf trace -e syscall command        # Specific syscall

# System-wide
perf record -a -g -- sleep 10        # Record all CPUs for 10s
perf top -a                          # Live all CPUs
```

## Tips and Gotchas

### 1. Compile with Frame Pointers

Without frame pointers, stack traces may be incomplete:

```bash
# GCC
g++ -fno-omit-frame-pointer -O2 program.cpp

# Or use DWARF unwinding
perf record --call-graph dwarf ./program
```

### 2. Install Debug Symbols

For system libraries:

```bash
# Ubuntu/Debian
sudo apt install libc6-dbg

# Fedora
sudo dnf debuginfo-install glibc
```

### 3. Kernel Symbols

```bash
# Check if available
cat /proc/kallsyms | head

# If restricted
sudo sysctl -w kernel.kptr_restrict=0
```

### 4. Beware of Sampling Bias

99 Hz sampling is usually enough. Higher frequencies don't always mean better data - they can introduce bias and overhead.

### 5. Profile Representative Workloads

Profile with realistic data. A profile of sorting 100 elements tells you nothing about sorting 1 million elements.

### 6. Multiple Runs

Hardware counters can vary. Run multiple times:

```bash
perf stat -r 5 ./program  # Run 5 times, show stats
```

## GUI Alternatives

If you prefer GUIs:

- **Hotspot**: KDE tool for viewing perf data. `sudo apt install hotspot`
- **Firefox Profiler**: Upload `perf script` output to profiler.firefox.com
- **Speedscope**: Web-based flame graph viewer at speedscope.app

```bash
# For Hotspot
perf record -g ./program
hotspot perf.data

# For Firefox Profiler
perf script -F +pid > profile.txt
# Upload to https://profiler.firefox.com
```

## Summary

```
┌─────────────────────────────────────────────────────────────────┐
│ Task                        │ Command                          │
├─────────────────────────────┼──────────────────────────────────┤
│ Quick CPU stats             │ perf stat ./program              │
│ Where is CPU time spent?    │ perf record -g ./program         │
│                             │ perf report                      │
│ Cache miss analysis         │ perf stat -e cache-misses ./prog │
│ Where are cache misses?     │ perf record -e cache-misses -g   │
│ Branch prediction issues    │ perf stat -e branch-misses ./pr  │
│ Generate flame graph        │ perf script | stackcollapse |    │
│                             │ flamegraph.pl > out.svg          │
│ Real-time monitoring        │ perf top -p PID                  │
│ Trace syscalls              │ perf trace ./program             │
└─────────────────────────────┴──────────────────────────────────┘
```

Start with `perf stat` to understand the overall picture. Use `perf record` + `perf report` to find hotspots. Generate flame graphs for complex call stacks.

The bottleneck is always somewhere. `perf` helps you find it.

## References

### Official Documentation

- [Kernel Perf Wiki](https://perf.wiki.kernel.org/index.php/Tutorial) - Official tutorial
- [perf man pages](https://man7.org/linux/man-pages/man1/perf.1.html) - Complete reference

### Brendan Gregg's Resources

- [Linux perf Examples](https://www.brendangregg.com/perf.html) - Essential one-liners and examples
- [CPU Flame Graphs](https://www.brendangregg.com/FlameGraphs/cpuflamegraphs.html) - Flame graph guide
- [FlameGraph Tools](https://github.com/brendangregg/FlameGraph) - Scripts for generating flame graphs
- [Linux Performance](https://www.brendangregg.com/linuxperf.html) - Overview of Linux performance tools

### Tutorials

- [Baeldung: Analyzing Cache Misses](https://www.baeldung.com/linux/analyze-cache-misses) - Cache analysis tutorial
- [Sand Software Sound: perf Tutorial](http://sandsoftwaresound.net/perf/perf-tutorial-hot-spots/) - Finding hot spots
- [Red Hat: Getting Started with perf](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/8/html/monitoring_and_managing_system_status_and_performance/getting-started-with-perf_monitoring-and-managing-system-status-and-performance)

### GUI Tools

- [KDAB Hotspot](https://github.com/KDAB/hotspot) - Linux perf GUI
- [Firefox Profiler](https://profiler.firefox.com/) - Web-based profile viewer
- [Speedscope](https://www.speedscope.app/) - Flame graph visualizer

### Books

- [Systems Performance, 2nd Edition](https://www.brendangregg.com/systems-performance-2nd-edition-book.html) - Brendan Gregg's comprehensive book
- [BPF Performance Tools](https://www.brendangregg.com/bpf-performance-tools-book.html) - Advanced tracing with eBPF
