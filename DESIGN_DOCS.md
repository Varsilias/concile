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
