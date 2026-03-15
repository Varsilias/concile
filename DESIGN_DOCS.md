# Day 1 - 25th February, 2026
Most of today was spent learning how `flag.FlagSet` from the **flag** package works because, implementing a nice reusable command registration hook. I wanted to have the `subcommand` experience most Go CLI tools have without using the popular `pflag` package.

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
There are plenty that I already forsee but I would not want to get involved in premature optimisation until I start benchmarking operations, besides I have not implemented the main file processing logic.


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
Which means my run probably reached around:
```
~75–80M transactions
```
before I killed it.
# Day 5 - 11th March
Started today by implemeting something I thought about skipping but my mind could not go off it.
The WAl file stores `Reference IDs` as strings plus an extra `newline("\n")` character. So far wwe have seen the `SessionID` of transaction records to be the most unique key for detecting duplicate and each SessionID is a string that `30-bytes` long including the newline character we have `31-bytes` per transaction record.
For 10 Million Transaction Records, assuming the entire record has a unique Session ID, that givies us
```bash
10 Million Record * 31 bytes = 310 Million Bytes ≈ 300MB WAL Log file
```
## The Solution
I converted every SessionID string into a fixed `8-byte` binary encoding representation. Before that, you need to know that you cannot straight up convert a **string** to **binary**, I had to convert the string to and unsigned 64 bit compatible integer and then converted that to 8-byte binary.
With the new implementation, for 10 Million Records, we should then have
```bash
10 Million Record * 8 bytes = 80 Million Bytes ≈ 80MB WAL Log file
```
Percentage difference
```bash
310/80 * 100%
```
We gain approximately 80% disk space back which can be used to store something else. The best part, we do all this without reducing performance. Infact, after some profiling, we gained in speed.
Here are some numbers:
_For 1 Million records we are roughly at the same speed(3s) as before when no duplicates were detected_
```bash
⏱️  WAL Replay took 1.077167ms

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-11T13:14:12.397+01:00
Ended At:       2026-03-11T13:14:12.399+01:00
Duration:       1.127583ms

==================================================
Processed 271.61 MiB of data
⏱️  Transaction Processor took 3s

==============================
       INGESTION REPORT       
==============================
Processed:      1000000
Failed:         0
Duplicates:     0
Duration:       3s

==============================

```

But for duplicates and rebuilding the log before processing, we hover between `60` & `80` milliseconds for replay time but we are consistently below `2s` in processing time
```bash
⏱️  WAL Replay took 83.783334ms

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-11T13:15:13.909+01:00
Ended At:       2026-03-11T13:15:13.993+01:00
Duration:       83.8ms

==================================================
Processed 271.61 MiB of data
⏱️  Transaction Processor took 1.99s

==============================
       INGESTION REPORT       
==============================
Processed:      0
Failed:         0
Duplicates:     1000000
Duration:       1.99s

==============================


⏱️  WAL Replay took 61.834333ms

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-11T13:15:27.365+01:00
Ended At:       2026-03-11T13:15:27.427+01:00
Duration:       61.853042ms

==================================================
Processed 271.61 MiB of data
⏱️  Transaction Processor took 1.98s

==============================
       INGESTION REPORT       
==============================
Processed:      0
Failed:         0
Duplicates:     1000000
Duration:       1.98s

==============================
```
I also did some **CPU** and **Memory** Profiling

### Memory Profile
```bash
go tool pprof mem.prof
File: main
Type: inuse_space
Time: 2026-03-11 13:36:36 WAT
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 2722.32kB, 100% of 2722.32kB total
Showing top 10 nodes out of 16
      flat  flat%   sum%        cum   cum%
 1184.27kB 43.50% 43.50%  1184.27kB 43.50%  runtime/pprof.StartCPUProfile
    1026kB 37.69% 81.19%     1026kB 37.69%  runtime.mallocgc
  512.05kB 18.81%   100%   512.05kB 18.81%  time.Sleep
         0     0%   100%  1184.27kB 43.50%  main.main
         0     0%   100%     1026kB 37.69%  runtime.allocm
         0     0%   100%  1184.27kB 43.50%  runtime.main
         0     0%   100%     1026kB 37.69%  runtime.mstart
         0     0%   100%     1026kB 37.69%  runtime.mstart0
         0     0%   100%     1026kB 37.69%  runtime.mstart1
         0     0%   100%     1026kB 37.69%  runtime.newm
```

### CPU Profile
```bash
go tool pprof cpu.prof
File: main
Type: cpu
Time: 2026-03-11 13:40:49 WAT
Duration: 21.71s, Total samples = 19.42s (89.47%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 12930ms, 66.58% of 19420ms total
Dropped 162 nodes (cum <= 97.10ms)
Showing top 10 nodes out of 101
      flat  flat%   sum%        cum   cum%
    3560ms 18.33% 18.33%     3560ms 18.33%  syscall.rawsyscalln
    2330ms 12.00% 30.33%     2420ms 12.46%  runtime.mapaccess2_fast64
    1920ms  9.89% 40.22%     1920ms  9.89%  encoding/json.stateInString
    1100ms  5.66% 45.88%     3340ms 17.20%  encoding/json.checkValid
    1000ms  5.15% 51.03%     1200ms  6.18%  encoding/json.unquoteBytes
     930ms  4.79% 55.82%      930ms  4.79%  runtime.madvise
     730ms  3.76% 59.58%      790ms  4.07%  encoding/json.(*decodeState).rescanLiteral
     530ms  2.73% 62.31%      530ms  2.73%  runtime.pthread_cond_signal
     430ms  2.21% 64.52%      430ms  2.21%  runtime.memclrNoHeapPointers
     400ms  2.06% 66.58%      750ms  3.86%  runtime.mapassign_fast64
```

The Profiling data tells us something very important.

The Top CPU Consumers are
```bash
18% syscall.rawsyscalln 
12% runtime.mapaccess2_fast64
9% encoding/json.stateInString
5% encoding/json.checkValid
5% encoding/json.unquoteBytes
```

`Concile` is spending more time in exactly **three places:**
1. disk IO - syscall.rawsyscalln
2. map operations - runtime.mapaccess2_fast64(this used to be runtime.mapaccess2_faststr)
3. JSON parsing - encoding/json.*

Memory shows that for 10 Million records with 8MB of existing WAL file, we use a total of
```bash
  ~2.7MB
```
Which is ridiculously small and the biggest consumer is not even my code logic itself, it is the code snippet that collect CPU profiling data
```bash
runtime/pprof.StartCPUProfile
```

# Day 6 - 12th March
Yesterday as part of the binary state representation implementation, I also wrote a first-pass implementation for the concurrent processing of data we ingest. The idea is to introduce concurrency via a CLI flag `--workers` and with that we configure how many goroutines we will spin up and then concurrently process the file.
My first implementation turned out to be slower than sequential processing even though we ran about **10 goroutines**. You might ask qhy I started my test with 10. It is because the `--worker` flag is optional and defaults to the number of CPUs present in the host machine and mine is a 10-core Apple Macbook Air.
The sequential processing on 10 million records took around **25-30** seconds but my first pass implementation took **55 second** on 10 goroutines to process the same 10 million records which is God awful for a concurrent processing system that should actually speed things up.
Just like I have been doing all this while, I implement, test, collect metrics and observe to see what the issues my be.
### Here are some of my observations
1. **Memory Stayed the same:** the same 2.7MB
```bash
File: main
Type: inuse_space
Time: 2026-03-11 17:43:18 WAT
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 2754.02kB, 100% of 2754.02kB total
Showing top 10 nodes out of 20
      flat  flat%   sum%        cum   cum%
 1184.27kB 43.00% 43.00%  1184.27kB 43.00%  runtime/pprof.StartCPUProfile
  544.67kB 19.78% 62.78%   544.67kB 19.78%  github.com/xuri/excelize/v2.init
     513kB 18.63% 81.41%      513kB 18.63%  runtime.mallocgc
  512.08kB 18.59%   100%   512.08kB 18.59%  compress/gzip.NewWriterLevel
         0     0%   100%  1184.27kB 43.00%  main.main
         0     0%   100%      513kB 18.63%  runtime.allocm
         0     0%   100%   544.67kB 19.78%  runtime.doInit (inline)
         0     0%   100%   544.67kB 19.78%  runtime.doInit1
         0     0%   100%  1728.94kB 62.78%  runtime.main
         0     0%   100%      513kB 18.63%  runtime.mstart
```
2. **CPU Time increased significantly:** I saw over 60% increase from dominated mainly by waiting time.
The dominant factor before concurrent processing was `syscalls` as shown here
```bash
File: main
Type: cpu
Time: 2026-03-11 13:40:49 WAT
Duration: 21.71s, Total samples = 19.42s (89.47%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 12930ms, 66.58% of 19420ms total
Dropped 162 nodes (cum <= 97.10ms)
Showing top 10 nodes out of 101
      flat  flat%   sum%        cum   cum%
    3560ms 18.33% 18.33%     3560ms 18.33%  syscall.rawsyscalln
    2330ms 12.00% 30.33%     2420ms 12.46%  runtime.mapaccess2_fast64
    1920ms  9.89% 40.22%     1920ms  9.89%  encoding/json.stateInString
    1100ms  5.66% 45.88%     3340ms 17.20%  encoding/json.checkValid
    1000ms  5.15% 51.03%     1200ms  6.18%  encoding/json.unquoteBytes
     930ms  4.79% 55.82%      930ms  4.79%  runtime.madvise
     730ms  3.76% 59.58%      790ms  4.07%  encoding/json.(*decodeState).rescanLiteral
     530ms  2.73% 62.31%      530ms  2.73%  runtime.pthread_cond_signal
     430ms  2.21% 64.52%      430ms  2.21%  runtime.memclrNoHeapPointers
     400ms  2.06% 66.58%      750ms  3.86%  runtime.mapassign_fast64
```
It became threading and sleep signals after concurrent implementation.
```bash
File: main
Type: cpu
Time: 2026-03-11 17:43:18 WAT
Duration: 55.23s, Total samples = 87.30s (158.08%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 86.06s, 98.58% of 87.30s total
Dropped 135 nodes (cum <= 0.44s)
Showing top 10 nodes out of 44
      flat  flat%   sum%        cum   cum%
    46.80s 53.61% 53.61%     46.80s 53.61%  runtime.pthread_cond_wait
    27.72s 31.75% 85.36%     27.72s 31.75%  syscall.rawsyscalln
     7.36s  8.43% 93.79%      7.36s  8.43%  runtime.pthread_cond_signal
     4.13s  4.73% 98.52%      4.13s  4.73%  runtime.usleep
     0.01s 0.011% 98.53%     27.45s 31.44%  internal/poll.(*FD).Write
     0.01s 0.011% 98.55%     46.82s 53.63%  runtime.notesleep
     0.01s 0.011% 98.56%     46.81s 53.62%  runtime.semasleep
     0.01s 0.011% 98.57%      3.52s  4.03%  runtime.stealWork
     0.01s 0.011% 98.58%      4.49s  5.14%  runtime.systemstack
         0     0% 98.58%     27.55s 31.56%  github.com/Varsilias/concile/internal/persistence.(*MemoryStore).Record
```
There was also significant time spend on access our im-memory map, and I guess that because our workers were so fast due to them doing very little work, they spend most of their time contending to access the in-memory map which is guarded by a `sync.Mutex`.

3. **The Workers Performed Very Little Tasks:** the data that shows the actual bottleneck is my manual telemetry tracking. I logged the capacity of the channel every *2 seconds* so I could see what is going on inside the queue. My implementation at this point uses one channel shared by all workers. The data shows that the workers were finishing tasks so fast that there was barely ever any data inside the channel whenever it logged. Essentially, my implementation introduced a negative backpressure and that cause workers to spend more time either *"waiting"* or *"sleeping"*.
```bash
worker count not set, defaulting to total number of CPU cores present 10
⏱️  WAL Replay took 715.916µs

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-11T17:43:18.076+01:00
Ended At:       2026-03-11T17:43:18.077+01:00
Duration:       736.083µs

==================================================
2026/03/11 17:43:20 Current Backlog: 0/10000
2026/03/11 17:43:22 Current Backlog: 0/10000
2026/03/11 17:43:24 Current Backlog: 0/10000
2026/03/11 17:43:26 Current Backlog: 0/10000
2026/03/11 17:43:28 Current Backlog: 0/10000
2026/03/11 17:43:30 Current Backlog: 0/10000
2026/03/11 17:43:32 Current Backlog: 0/10000
2026/03/11 17:43:34 Current Backlog: 0/10000
2026/03/11 17:43:36 Current Backlog: 0/10000
2026/03/11 17:43:38 Current Backlog: 0/10000
2026/03/11 17:43:40 Current Backlog: 0/10000
2026/03/11 17:43:42 Current Backlog: 0/10000
2026/03/11 17:43:44 Current Backlog: 0/10000
2026/03/11 17:43:46 Current Backlog: 0/10000
2026/03/11 17:43:48 Current Backlog: 0/10000
2026/03/11 17:43:50 Current Backlog: 0/10000
2026/03/11 17:43:52 Current Backlog: 0/10000
2026/03/11 17:43:54 Current Backlog: 0/10000
2026/03/11 17:43:56 Current Backlog: 0/10000
2026/03/11 17:43:58 Current Backlog: 0/10000
2026/03/11 17:44:00 Current Backlog: 0/10000
2026/03/11 17:44:02 Current Backlog: 0/10000
2026/03/11 17:44:04 Current Backlog: 0/10000
2026/03/11 17:44:06 Current Backlog: 0/10000
2026/03/11 17:44:08 Current Backlog: 0/10000
2026/03/11 17:44:10 Current Backlog: 0/10000
2026/03/11 17:44:12 Current Backlog: 0/10000
Processed 2.65 GiB of data
⏱️  Transaction Processor took 55.1s

==============================
       INGESTION REPORT       
==============================
Processed:      10000000
Failed:         0
Duplicates:     0
Duration:       55.1s

==============================
```
The next time I hop on, I will implement a few optimisation plans I have

# Day 7 - 14th March
## Part One: 
Today I started implementing the optimisations I had in mind, starting with moving the logic for JSON encoding to the worker goroutines. Before today, the transformation was done in the main process and the transformed data is then sent to the channel. That implementation made my workers to starve and increased the time taken to process the data significantly.


After my implementation, I realised that the JSON encoding logic was corrupted and because of that the data was no longer being encoded properly, after investagtion i realised that my problem was a **Data Race Problem** caused by using `ReadSlice("\n")` method from the `bufio` package. 

It turned out the the underlying slice used in the method's implementation was always reused after each read which means that even though we passed the line that was read to the worker, because the speed of the worker and the main process are not the same, before the worker will wake up to process the line from the channel, the underlying slice would have either been overriden with newly read line either halfway or in full(mostly halfway), it cause the bytes to be corrupted and the JSON encoder could not recognise it as a valid JSON. I solved this by switching to `ReadBytes("\n") which gives the same effect but allocates and returns a new byte slice for each line read.
After the first implementation, there wasn't much change in performance, I only saw a 5 seconds reduction in time spent doing work and the CPU profile looks almost the same even slightly elevated.
## Part Two:
From previous benchmark I noticed that these 2 lines were at the top of the performance bottleneck
```bash
    46.80s 53.61% 53.61%     46.80s 53.61%  runtime.pthread_cond_wait
    27.72s 31.75% 85.36%     27.72s 31.75%  syscall.rawsyscalln
```
what does 2 lines mean is that much of the program is being dominated by `locks` which is placed on the in-memory map & `syscalls` when writing processed records keys to file. At the time were making `write` syscall for every 8 byte we want to write to file. After moving the major work(json dencoding) to the workers those 2 lines did not change much, this is the CPU profile after that
```bash
File: main
Type: cpu
Time: 2026-03-14 18:06:59 WAT
Duration: 49.71s, Total samples = 88.70s (178.44%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 87.93s, 99.13% of 88.70s total
Dropped 137 nodes (cum <= 0.44s)
Showing top 10 nodes out of 40
      flat  flat%   sum%        cum   cum%
    39.26s 44.26% 44.26%     39.26s 44.26%  runtime.pthread_cond_wait
    36.01s 40.60% 84.86%     36.01s 40.60%  syscall.rawsyscalln
     7.14s  8.05% 92.91%      7.14s  8.05%  runtime.usleep
     5.47s  6.17% 99.08%      5.47s  6.17%  runtime.pthread_cond_signal
     0.03s 0.034% 99.11%      6.66s  7.51%  runtime.stealWork
     0.01s 0.011% 99.12%     36.09s 40.69%  github.com/Varsilias/concile/internal/processor.worker
     0.01s 0.011% 99.13%      5.49s  6.19%  runtime.wakep
         0     0% 99.13%     35.74s 40.29%  github.com/Varsilias/concile/internal/persistence.(*MemoryStore).Record
         0     0% 99.13%     35.71s 40.26%  github.com/Varsilias/concile/internal/persistence.(*WAL).Append
         0     0% 99.13%     36.09s 40.69%  github.com/Varsilias/concile/internal/processor.Run.func1
```

My next move was to implement a batch writes to the WAL file instead of writing every **8-byte** to file for each line processed. When the worker calls `store.Record()` I no longer append to WAL file and update the in-memory map in one go, that has been replaced by writing the record to channel. Then we run a background go routine that **flushes** the channel based on
1. Size of the goroutine
2. Time - set to ever 1 second for now
3. When the process is cancelled
This implementation reduced the time spent doing work drastically. We went from processing **10 Million** records in
1. **30 seconds** sequentially on a single core
2. **55 seconds** after first concurrent implementation
3. **50 seconds** when work was moved to worker
to procesing the same number of records anywhere between **7 seconds** and **22 seconds** on workers greater than **one**
Sample benchmarks
```bash
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_10M.jsonl --provider=Zbank --workers=2
⏱️  WAL Replay took 181.542µs

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-14T21:58:03.102+01:00
Ended At:       2026-03-14T21:58:03.102+01:00
Duration:       192.125µs

==================================================
2026/03/14 21:58:05 Current Backlog: 2000/2000
2026/03/14 21:58:07 Current Backlog: 2000/2000
2026/03/14 21:58:09 Current Backlog: 2000/2000
2026/03/14 21:58:11 Current Backlog: 2000/2000
2026/03/14 21:58:13 Current Backlog: 1994/2000
2026/03/14 21:58:15 Current Backlog: 2000/2000
2026/03/14 21:58:17 Current Backlog: 1991/2000
2026/03/14 21:58:19 Current Backlog: 2000/2000
2026/03/14 21:58:21 Current Backlog: 2000/2000
2026/03/14 21:58:23 Current Backlog: 1991/2000
2026/03/14 21:58:25 Current Backlog: 2000/2000
Processed 2.65 GiB of data
⏱️  Transaction Processor took 22.82s

==============================
       INGESTION REPORT       
==============================
Processed:      10000000
Failed:         0
Duplicates:     0
Duration:       22.82s

==============================
2026/03/14 21:58:25 WAL Writer shutting down...

# Second run while WAL file exists
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_10M.jsonl --provider=Vbank --workers=2
⏱️  WAL Replay took 1.14s

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-14T21:58:38.078+01:00
Ended At:       2026-03-14T21:58:39.214+01:00
Duration:       1.14s

==================================================
2026/03/14 21:58:41 Current Backlog: 2000/2000
2026/03/14 21:58:43 Current Backlog: 1998/2000
2026/03/14 21:58:45 Current Backlog: 1999/2000
2026/03/14 21:58:47 Current Backlog: 2000/2000
2026/03/14 21:58:49 Current Backlog: 2000/2000
Processed 2.65 GiB of data
⏱️  Transaction Processor took 12.27s

==============================
       INGESTION REPORT       
==============================
Processed:      2195
Failed:         0
Duplicates:     9997805
Duration:       12.27s

==============================
2026/03/14 21:58:50 WAL Writer shutting down...

# On 4 workers
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_10M.jsonl --provider=Vbank --workers=4
⏱️  WAL Replay took 162.208µs

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-14T21:59:19.777+01:00
Ended At:       2026-03-14T21:59:19.777+01:00
Duration:       192.958µs

==================================================
2026/03/14 21:59:21 Current Backlog: 4000/4000
2026/03/14 21:59:23 Current Backlog: 4000/4000
2026/03/14 21:59:25 Current Backlog: 4000/4000
2026/03/14 21:59:27 Current Backlog: 3999/4000
2026/03/14 21:59:29 Current Backlog: 4000/4000
2026/03/14 21:59:31 Current Backlog: 4000/4000
2026/03/14 21:59:33 Current Backlog: 4000/4000
2026/03/14 21:59:35 Current Backlog: 4000/4000
2026/03/14 21:59:37 Current Backlog: 4000/4000
2026/03/14 21:59:39 Current Backlog: 4000/4000
Processed 2.65 GiB of data
⏱️  Transaction Processor took 20.49s

==============================
       INGESTION REPORT       
==============================
Processed:      10000000
Failed:         0
Duplicates:     0
Duration:       20.49s

==============================
2026/03/14 21:59:40 WAL Writer shutting down...

# Second run on 4 workers
go run main.go ingest --file=~/Projects/personal/concile/data/inflow_10M.jsonl --provider=Vbank --workers=4
⏱️  WAL Replay took 1.19s

==================================================
       WAL REPLAY REPORT       
==================================================
Started At:     2026-03-14T21:59:45.860+01:00
Ended At:       2026-03-14T21:59:47.051+01:00
Duration:       1.19s

==================================================
2026/03/14 21:59:49 Current Backlog: 3971/4000
2026/03/14 21:59:51 Current Backlog: 3882/4000
2026/03/14 21:59:53 Current Backlog: 3951/4000
Processed 2.65 GiB of data
⏱️  Transaction Processor took 7.72s

==============================
       INGESTION REPORT       
==============================
Processed:      3770
Failed:         0
Duplicates:     9996230
Duration:       7.72s

==============================
2026/03/14 21:59:53 WAL Writer shutting down...

```
If you look at those logs, you will notice that we traded a little bit of consistency based on this finding especially when there is an existing *WAL* file to rebuild in-memory map from. We expect that at least less than or equal to **1000 * worker_count** will be reprocessed if the file containing that record has been processed before but this is not much of an issue as we have a strong **idempotency** implementation.


The CPU profile looks great too
```bash
File: main
Type: cpu
Time: 2026-03-14 21:59:45 WAT
Duration: 7.91s, Total samples = 28.64s (361.85%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top 
Showing nodes accounting for 26.53s, 92.63% of 28.64s total
Dropped 141 nodes (cum <= 0.14s)
Showing top 10 nodes out of 87
      flat  flat%   sum%        cum   cum%
    16.86s 58.87% 58.87%     16.86s 58.87%  runtime.usleep
     4.16s 14.53% 73.39%      4.16s 14.53%  runtime.pthread_cond_wait
     3.35s 11.70% 85.09%      3.35s 11.70%  runtime.pthread_cond_signal
     0.72s  2.51% 87.60%      0.72s  2.51%  runtime.madvise
     0.42s  1.47% 89.07%      0.81s  2.83%  runtime.mapassign_fast64
     0.39s  1.36% 90.43%      0.39s  1.36%  syscall.rawsyscalln
     0.22s  0.77% 91.20%      0.22s  0.77%  encoding/json.stateInString
     0.15s  0.52% 91.72%      0.15s  0.52%  encoding/json.unquoteBytes
     0.15s  0.52% 92.25%      0.15s  0.52%  runtime.memclrNoHeapPointers
     0.11s  0.38% 92.63%      0.39s  1.36%  encoding/json.checkValid
```
You can see the *syscall* that is **syscall.rawsyscalln**  has reduced drastically and no longer a bottlenech due to the batch implementation.
The current bottleneck which exists now is because of the locks used to prevent concurrent map access.
I have an implementation in mind but need to reason about it.
See you tomorrow