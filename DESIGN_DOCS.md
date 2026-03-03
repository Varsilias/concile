# Day 1 - 25th February, 2026
Most of today was spent learning how `flag.FlagSet` from the **flag** package works because, implementing a nice reusable command registration hook. I wanted to have the `subcommand` experience that most Go CLI tools have without using the popular `pflag` package.

I also ended up wiring up a command to convert `xlsx` file into `jsonl`

I spent a considerable amount of time deciding on the shape of transaction data and considerations for handling different fields I expect to be present in the datasource. While most fields ended up being string, there were consideration on my end to ensure that things move smoothly as I progress
1. How would `dates` be represented
2. How would `money` be represented
3. How should `duplicate` records be handled

For the first 2 questions, I decided to have 2 different structs and convert from one to the other. I have a `RawTransaction` struct which represents the Excel file structure and intentionally made most things as string, we then have a `CanonicalTransaction` struct which will be the main structure we will be working with after converting to it.

For Duplicate handling, I have chosen to
- Keep the first encountered record
- Log(Warn) for duplicates detected after
- Record the number of duplicates as encountered during processing for statistics

## Performance Considerations
There are plenty that I already forsee but I would not want to get involved in premature optimisation until I start benchmarking operations, besides I have not implemented the main file processing.


Here are some performance considerations to think of
- Would performance be affected when we do the conversion from `raw` to `canonical` struct before processing and would there be gains if we do not convert
- How do we avoid loading entire file into memory
- The goal is to process 100k lines of JSONL financial records in 3 seconds, what can we do to achieve that?
- JSON serialisation has always been a bottleneck in modern systems, how do I avoid that?

# Day 2 - 28th February, 2026
I used today to write the core logic for transaction record handling. I ended wiring up the Statistis tracking and Bytes Size conversion to human readable format. Which now led to `stdout` like so:
```bash
Processed 27.16 MiB of data
⏱️  Transaction Processor took 169.749042ms

==============================
       INGESTION REPORT       
==============================
Processed:      100000
Failed:         0
Duplicates:     0
Duration:       169.751167ms

==============================
```

and this

```bash
Processing [inflow] -> /Users/danielokoronkwo/Projects/personal/concile/data/inflow.jsonl
Processing [outflow] -> /Users/danielokoronkwo/Projects/personal/concile/data/outflow.jsonl
⏱️  XLSX Conversion took 2.79s
```

## Performance Consideration
- **os.Open:** I now have a much clearer understanding and insight why you should use `os.Open or os.OpenFile` over `os.Read, os.ReadFile` when dealing with high performance applications. It used to be confusing a bit but it is clearer now. With `os.Open` you create a sort of point(called file descriptor) to the file and then you can read the records in the file line by line and that way you never have to worry about running out of memory. With `os.Read` you are loading the entire file content into memory before reading, which may be fine for few `KiloBytes` of data file, but GigaBytes of data record will surely fail. With `os.Open` memory is almost the same whether you are processing kilobytes of data or PetaBytes of data
- **Scanner or Reader:** Based on my research, `Reader` should be the preferred way for performing line-by-line file processing as it gives you performance and control. But most times, the control means you have to do a lot of manual labour yourself like ensuring that the line returned to you after you call one of its method `ReadString, ReadLine, ReadByte` does not contain the delimiter itself. The `Scanner` struct simplify things and prevents you from the manual labour like in `Reader` but it comes at a cost. It has an internal buffer size of  `4096 bytes` for a start and grows to `bufio.MaxScanTokenSize(64KB)` which maybe a limitation if the size of each line of the file you are processing is larger than 64KB
- **Zero Allocation:** Initially, I used the `ReadString` method of the `Reader` Type, until I learnt about the implication of the function. It allocate new string in memory every single time, for as many number of lines there are in the file. In Performance oriented programming, you already know that constant allocation is not memory efficient especially for a garbage collected language as that will cause the GC to do more work which leads to GC Pauses. I found that the alternative `ReadByte` and `ReadSlice` are better because they maintain a pointer to one memory location and update that memory address on every loop until the entire file is processed. This prevents unneceaary allocation and increases cache friendlines

## Powerful Insights for today
- Ingested 100k JSONL file and did it in an average of **180ms**, that's insane because the goal for now is 100k in under `3s`. I still need to make some conversion, which I bet increases the time taken.
- Once the full implementation is in place, I will write a benchmark test for automated benchmarking

# Day 3 - 3rd March, 2026
Today was for implementing fully the Normalisation from `RawTransaction` Record struct to `CanonicalTransaction`. I left this initially because the implementation in my head felt like I was going to write too much code and that did not feel normal. I eventually implemented it today and let's just say it had a lot of If-Else statements in there which still feels weird to me but hey this is the Go world, you have to be explicit.

After implemetation, I had to do a lot of testing and profiling. For my manual test, I initially saw my number go from `160+ms` on average to `600ms` on average still on 100k records and while it does not scream too much, I felt uneasy and wanted to explain it away with all the parsing and conversion I performed in the Normalisation function. Apparently, the first version of the function implementation was failing and that cause the increase in time spent. After correction we came back to `200ms` average, which is great.

I then went into Profiling and Benchmarking Test. I Profiled the CPU usage to see what is going on. I added the following snippet to my entry file
```go
f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
```
This snippet override create a `cpu.prof` file after every run which you use to run analysis on how the program performs in terms of CPU usage. The command to trigger the profiling is
`go tool pprof cpu.prof`

This command shows an interactive shell-like prompt, if you enter thr prompt `top` and execute, it gives you output similar to what you see below

Here is the first ever result. 
```zsh
File: main
Type: cpu
Time: 2026-03-03 18:05:50 WAT
Duration: 1.61s, Total samples = 320ms (19.82%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top 
Showing nodes accounting for 320ms, 100% of 320ms total
Showing top 10 nodes out of 48
      flat  flat%   sum%        cum   cum%
     290ms 90.62% 90.62%      290ms 90.62%  syscall.rawsyscalln
      10ms  3.12% 93.75%       10ms  3.12%  runtime.memclrNoHeapPointers
      10ms  3.12% 96.88%       10ms  3.12%  runtime.pthread_cond_wait
      10ms  3.12%   100%       10ms  3.12%  runtime.typePointers.nextFast
         0     0%   100%       30ms  9.38%  bufio.(*Reader).ReadSlice
         0     0%   100%       30ms  9.38%  bufio.(*Reader).fill
         0     0%   100%      300ms 93.75%  github.com/Varsilias/concile/internal/processor.Run
         0     0%   100%      300ms 93.75%  github.com/Varsilias/concile/internal/processor.init.0.func2
         0     0%   100%       10ms  3.12%  github.com/Varsilias/concile/internal/processor.reconcile
         0     0%   100%       30ms  9.38%  internal/poll.(*FD).Read
(pprof) %           
```

After some explanation from my buddy ChatGPT, I got to know that we did relative okay, but the `90.62%` in the first result, shows that the bottleneck albeit not much overall is `syscall`.
In simple terms, they problem is that we spend most of our processing time waiting for the CPU to respond to disk read requests. This is goodnews because our computation is not heavy per say, but the syscall is using more time that should be happening.

> Essentially, our bottleneck is not Compute related but IO-bound

**The Solution:** Is a general sense of things, if you want to increase thoroughput for any batch processing system, typically you increase the size of each batch to be processed. The `NewReader` of the `bufio` package has an internal batch limit of `4KB`, I had to switch to `NewReaderSize` which allows to customise the size of the internal batch like so `buffer := bufio.NewReaderSize(f, 1<<20)`, I increased it to 1MB with bit shifting. Since we are doing more context switching as a result with each syscalls, why not reduce the number of times we have to ask. For context, the file being processed was `27MB` which was around
```
27MB / 4KB ≈ 6912 read syscalls
```
After the change, here is the new number
```
27MB / 1MB ≈ 27 read syscalls
```

~7000 syscalls to ~27. By now I believe you get the idea. Here is the new result

```zsh
File: main
Type: cpu
Time: 2026-03-03 19:06:06 WAT
Duration: 404.35ms, Total samples = 210ms (51.94%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 190ms, 90.48% of 210ms total
Showing top 10 nodes out of 81
      flat  flat%   sum%        cum   cum%
      50ms 23.81% 23.81%       50ms 23.81%  syscall.rawsyscalln
      30ms 14.29% 38.10%       30ms 14.29%  runtime.madvise
      20ms  9.52% 47.62%       20ms  9.52%  encoding/json.unquoteBytes
      20ms  9.52% 57.14%       20ms  9.52%  runtime.(*mspan).init
      20ms  9.52% 66.67%       20ms  9.52%  runtime.pthread_cond_signal
      10ms  4.76% 71.43%       10ms  4.76%  encoding/json.stateBeginStringOrEmpty
      10ms  4.76% 76.19%       10ms  4.76%  encoding/json.stateEndValue
      10ms  4.76% 80.95%       10ms  4.76%  runtime.memclrNoHeapPointers
      10ms  4.76% 85.71%       10ms  4.76%  runtime.mmap
      10ms  4.76% 90.48%       10ms  4.76%  runtime.pthread_cond_wait
```

I also ran some BenchMark Tests on the Normalisation function and here is the result
```zsh
goos: darwin
goarch: arm64
pkg: github.com/Varsilias/concile/internal/pkg
cpu: Apple M4
BenchmarkNormalizeInflow-10     	 7419302	       140.5 ns/op	      16 B/op	       1 allocs/op
BenchmarkNormalizeOutflow-10    	11684474	       103.3 ns/op	       0 B/op	       0 allocs/op
PASS
```
Essential for outflow transaction types, we have 0 allocation per operation, which means in simple terms that the Garbage Collector will not have any work to do even if we ran the function on 1 Million json records. But there is 16 Byte allocation per operation for Inflow records which I am yet to figure out where it is happening. My guts says it has something to do with some kind of `string` operation I am doing but hey this is where I stopped for now.

I ran a memory profiling on the InflowBenchMark Test and the result

Running the test
```zsh
 go test -bench=BenchmarkNormalizeInflow ./internal/pkg -benchmem -memprofile mem.prof

goos: darwin
goarch: arm64
pkg: github.com/Varsilias/concile/internal/pkg
cpu: Apple M4
BenchmarkNormalizeInflow-10    	 7276989	       142.6 ns/op	      16 B/op	       1 allocs/op
PASS
ok  	github.com/Varsilias/concile/internal/pkg	2.656s
```

The Profiling
```zsh
File: pkg.test
Type: alloc_space
Time: 2026-03-03 18:45:50 WAT
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 119MB, 100% of 119MB total
Dropped 10 nodes (cum <= 0.60MB)
Showing top 10 nodes out of 18
      flat  flat%   sum%        cum   cum%
     117MB 98.32% 98.32%      117MB 98.32%  internal/bytealg.MakeNoZero
       2MB  1.68%   100%        2MB  1.68%  runtime.mallocgc
         0     0%   100%      117MB 98.32%  github.com/Varsilias/concile/internal/pkg.BenchmarkNormalizeInflow
         0     0%   100%      117MB 98.32%  github.com/Varsilias/concile/internal/pkg.Normalize
         0     0%   100%        1MB  0.84%  runtime.(*scavengerState).sleep
         0     0%   100%        1MB  0.84%  runtime.(*timer).maybeAdd
         0     0%   100%        1MB  0.84%  runtime.(*timer).modify
         0     0%   100%        1MB  0.84%  runtime.(*timer).reset (inline)
         0     0%   100%        1MB  0.84%  runtime.(*timers).addHeap
         0     0%   100%     1.50MB  1.26%  runtime.bgscavenge
```

Here is my personal telemetry record after today's implementation
```bash
go run main.go ingest --file=~/Projects/personal/concile/data/inflow.jsonl
Processed 27.16 MiB of data
⏱️  Transaction Processor took 205.727166ms

==============================
       INGESTION REPORT       
==============================
Processed:      100000
Failed:         0
Duplicates:     0
Duration:       205.729625ms

==============================
```