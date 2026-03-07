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

# Day 4 - 7th March, 2026
Today was supposed to be the start of stage 2 which is **Idempotency + Deduplication Engine**. But I thought it will be nice to manually stress test the existing logic. I introduced a python script for generating synthetic data following the known schema. I generate 1 Million records and then 10 Million record files and tried ingesting them.

### Here are some records:

**1 Million Records**
```bash 
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_1M.jsonl

Processed 271.60 MiB of data
⏱️  Transaction Processor took 2.27s

==============================
       INGESTION REPORT       
==============================
Processed:      999992
Failed:         0
Duplicates:     8
Duration:       2.27s

==============================
```

_CPU Profile for 1 Million Record_
```bash
go tool pprof cpu.prof
File: main
Type: cpu
Time: 2026-03-07 13:46:42 WAT
Duration: 2.42s, Total samples = 2.48s (102.62%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 1910ms, 77.02% of 2480ms total
Dropped 44 nodes (cum <= 12.40ms)
Showing top 10 nodes out of 121
      flat  flat%   sum%        cum   cum%
     780ms 31.45% 31.45%     1800ms 72.58%  github.com/Varsilias/concile/internal/processor.Run
     290ms 11.69% 43.15%      290ms 11.69%  runtime.madvise
     220ms  8.87% 52.02%      250ms 10.08%  runtime.scanObject
     190ms  7.66% 59.68%      230ms  9.27%  runtime.mapaccess2_faststr
     150ms  6.05% 65.73%      150ms  6.05%  syscall.rawsyscalln
      60ms  2.42% 68.15%       60ms  2.42%  runtime.memmove
      60ms  2.42% 70.56%       70ms  2.82%  runtime.tryDeferToSpanScan
      60ms  2.42% 72.98%       60ms  2.42%  runtime.usleep
      50ms  2.02% 75.00%      110ms  4.44%  encoding/json.checkValid
      50ms  2.02% 77.02%       50ms  2.02%  runtime.memclrNoHeapPointers
(pprof)
```

**Explanation**
From the *Ingestion Report*, we see that the timing report stays consistent. We had a median processing time for 100k records at `200+ms` which means that if we 10x our records, we also expect 10x time which is why we have `2+s`.
The Profiler shows a different output though especially for Inflow Records. If you read the report for the day before today, you will see where we had 16 Bytes of allocation every time we call `Normalize` on an Inflow Record, it turns out that at scale, it becomes a bottle neck as proven by this 2 lines:
```bash
     290ms 11.69% 43.15%      290ms 11.69%  runtime.madvise
     220ms  8.87% 52.02%      250ms 10.08%  runtime.scanObject
```
These are times used by the Garbage Collector to scan for Object that needs to be `removed(garbage collected)`. It was not very pronounced at 100k but when the number of records increase, then we awaken the GC. Also, our in-memory map for handling and detecting duplicate references also starts becoming a bottleneck at scale based on this line
```bash
     190ms  7.66% 59.68%      230ms  9.27%  runtime.mapaccess2_faststr
```

**10 Million Records**
```bash
Processed 2.65 GiB of data
⏱️  Transaction Processor took 28.23s

==============================
       INGESTION REPORT       
==============================
Processed:      9999574
Failed:         0
Duplicates:     426
Duration:       28.23s

==============================
```

_CPU Profile for 10 Million Record_
```bash
go tool pprof cpu.prof
File: main
Type: cpu
Time: 2026-03-07 13:51:09 WAT
Duration: 28.40s, Total samples = 35.92s (126.49%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 29290ms, 81.54% of 35920ms total
Dropped 151 nodes (cum <= 179.60ms)
Showing top 10 nodes out of 91
      flat  flat%   sum%        cum   cum%
    5380ms 14.98% 14.98%    21730ms 60.50%  github.com/Varsilias/concile/internal/processor.Run
    5350ms 14.89% 29.87%     8990ms 25.03%  runtime.scanObject
    3650ms 10.16% 40.03%     3670ms 10.22%  runtime.tryDeferToSpanScan
    3530ms  9.83% 49.86%     3530ms  9.83%  runtime.memclrNoHeapPointers
    3120ms  8.69% 58.55%     3120ms  8.69%  runtime.madvise
    2380ms  6.63% 65.17%     4780ms 13.31%  runtime.mapaccess2_faststr
    2380ms  6.63% 71.80%     2380ms  6.63%  runtime.memequal
    1520ms  4.23% 76.03%     1520ms  4.23%  runtime.usleep
    1120ms  3.12% 79.15%     1120ms  3.12%  runtime.memmove
     860ms  2.39% 81.54%      860ms  2.39%  runtime.(*spanInlineMarkBits).init
(pprof) 
```

## Stage 2
For this stage, the goal is to add a more robust `idempotency check`. **Stage 1** used an in-memory map alone to track duplicate items which mean we have to reprocess records again for every time we ingest a file.
### The Implemetation
The current implementation borrows from the idea that powers most popular storage engines. we track in-memory and flush to disk at intervals just like WAL(Write Ahead Logs) in Postgres and LSM-Trees. But my implementation for now does not include periodic flush to disk, instead we write to disk immediately we `Normalize` a record. We still maintain an `in-memory` map for quick lookup to check if a normalised record is a duplicate item. But once the ingestion finishes, the in-memory object is thrown away which is safe to do because we are also append to a `wal.log` file as we process.

Now, when a new ingestion command runs, we rebuild the in-memory object from the durable `wal.log` file, that way, every `ingest` command stays idempotent

#### Performance Consideration
1. Appending to file immediately after processing - for this, there was no peformance bottleneck and this is because we are perfoming sequential writes and not random writes
2. We are currently using 1 large file for WAL, what happens when we append `100M` keys - well I have not benchmarked `100M` keys yet but I have already seen some numbers for replay time alone and it is not looking good as data grows. I may have to introduce some encoding to make things smaller
3. What does startup time look like especially as log file grows - right now, it grows linearly as he file grows

**Some Numbers**

_For 100K Records_

**_Empty Log File_**
```bash
⏱️  WAL Replay took 90.292µs

========================================
       WAL REPLAY REPORT       
========================================
Started At:     2026-03-07T19:27:00.406+01:00
Ended At:       2026-03-07T19:27:00.406+01:00
Duration:       200.416µs

========================================
Processed 27.16 MiB of data
⏱️  Transaction Processor took 297.252333ms

==============================
       INGESTION REPORT       
==============================
Processed:      100000
Failed:         0
Duplicates:     0
Duration:       297.256417ms

==============================
```

**_After first run_**

```bash
⏱️  WAL Replay took 6.817459ms

========================================
       WAL REPLAY REPORT       
========================================
Started At:     2026-03-07T19:27:10.797+01:00
Ended At:       2026-03-07T19:27:10.804+01:00
Duration:       6.826958ms

========================================
Processed 27.16 MiB of data
⏱️  Transaction Processor took 193.783334ms

==============================
       INGESTION REPORT       
==============================
Processed:      0
Failed:         0
Duplicates:     100000
Duration:       193.786083ms

==============================
```

You can see that replay time increases as number of records in log increases. Imagine what it will look like for `1M,10M,100M` records.

I also simulated a crash and everything worked really well
```bash
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_1M.jsonl --provider=Vbank
^Csignal: interrupt
```
And the next run gave this

```bash
 go run main.go ingest --file=~/Projects/personal/concile/data/inflow_1M.jsonl --provider=Vbank
Processed 271.60 MiB of data
⏱️  Transaction Processor took 2.93s

==============================
       INGESTION REPORT       
==============================
Processed:      795188
Failed:         0
Duplicates:     204812
Duration:       2.93s

==============================
```
I pushed the ingestation pipeline to the limit with `100M` records which is about **28.48GB**, got to about `2.4GB` of log file but I had to kill the process half way.

If each key is about:
```bash
30 digits + newline ≈ 31 bytes
```
Then:
```bash
2.39GB / 31 ≈ ~77 million entries
```
Which means your run probably reached around:
```
~75–80M transactions
```
before I killed it.