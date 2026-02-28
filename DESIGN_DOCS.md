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