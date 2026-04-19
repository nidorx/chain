# Chain Framework - Benchmark Guide

Quick reference guide for running benchmarks in the Chain framework.

---

## Quick Start

### Run All Benchmarks

```bash
go test -bench=. -benchmem
```

This runs all 37 benchmarks and reports:
- Operations per second
- Nanoseconds per operation
- Bytes allocated per operation
- Number of allocations per operation

---

## Common Commands

### Run Specific Benchmark

```bash
# Run only static route lookup benchmarks
go test -bench=BenchmarkRouter_StaticRoute -benchmem

# Run only middleware benchmarks
go test -bench=BenchmarkMiddleware -benchmem

# Run only binding benchmarks
go test -bench=BenchmarkBinding -benchmem
```

### Run with Verbose Output

```bash
go test -bench=. -benchmem -v
```

### Run Specific Benchmark Category

```bash
# Route lookup benchmarks
go test -bench=BenchmarkRouter -benchmem

# Context benchmarks
go test -bench=BenchmarkContext -benchmem

# Full request benchmarks
go test -bench=BenchmarkFullRequest -benchmem
```

---

## Profiling

### CPU Profiling

```bash
# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof

# View profile
go tool pprof cpu.prof

# Common pprof commands:
#   top10        - Show top 10 functions
#   list FuncName - Show source for function
#   web          - Generate visual profile
```

### Memory Profiling

```bash
# Generate memory profile
go test -bench=. -memprofile=mem.prof

# View profile
go tool pprof mem.prof
```

### Block Profiling

```bash
# Generate block profile (contention)
go test -bench=. -blockprofile=block.prof

# View profile
go tool pprof block.prof
```

---

## Race Detection

```bash
# Run benchmarks with race detector
go test -bench=. -benchmem -race
```

**Note:** Race detection significantly impacts performance. Use for correctness checking, not performance measurement.

---

## Comparing Results

### Save Baseline

```bash
go test -bench=. -benchmem > baseline.txt
```

### After Changes

```bash
go test -bench=. -benchmem > current.txt
```

### Compare with benchstat

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Compare results
benchstat baseline.txt current.txt
```

---

## Benchmark Categories

### 1. Route Lookup
```bash
go test -bench=BenchmarkRouter_StaticRoute_Lookup -benchmem
go test -bench=BenchmarkRouter_ParameterRoute_Lookup -benchmem
go test -bench=BenchmarkRouter_WildcardRoute_Lookup -benchmem
```

### 2. Middleware
```bash
go test -bench=BenchmarkMiddleware_NoMiddleware -benchmem
go test -bench=BenchmarkMiddleware_MultipleMiddleware -benchmem
```

### 3. Context
```bash
go test -bench=BenchmarkContext_Creation -benchmem
go test -bench=BenchmarkContext_PoolRecycling -benchmem
```

### 4. Binding
```bash
go test -bench=BenchmarkBinding_JSON -benchmem
go test -bench=BenchmarkBinding_Query -benchmem
go test -bench=BenchmarkBinding_Form -benchmem
```

### 5. Full Request
```bash
go test -bench=BenchmarkFullRequest_StaticRoute -benchmem
go test -bench=BenchmarkFullRequest_JSONBinding -benchmem
```

### 6. Registration
```bash
go test -bench=BenchmarkRouteRegistration_Static -benchmem
go test -bench=BenchmarkRouteRegistration_Parameter -benchmem
```

### 7. Response
```bash
go test -bench=BenchmarkResponse_JSON -benchmem
go test -bench=BenchmarkResponse_Status -benchmem
```

---

## Tips

### 1. Run Multiple Times for Accuracy

```bash
# Run benchmark 3 times
go test -bench=. -benchmem -count=3
```

### 2. Control Benchmark Duration

```bash
# Run each benchmark for at least 10 seconds
go test -bench=. -benchmem -benchtime=10s
```

### 3. Filter Benchmarks

```bash
# Run only benchmarks matching pattern
go test -bench=JSON -benchmem

# Runs: BenchmarkBinding_JSON, BenchmarkFullRequest_JSONBinding, BenchmarkResponse_JSON
```

### 4. Parallel Benchmarks

```bash
# Run benchmarks in parallel (if multiple CPUs)
go test -bench=. -benchmem -parallel=4
```

---

## Interpreting Results

### Example Output

```
BenchmarkRouter_StaticRoute_100-8    1006806    1142 ns/op    1536 B/op    1 allocs/op
```

**Explanation:**
- `1006806` - Number of iterations completed
- `1142 ns/op` - Average time per operation
- `1536 B/op` - Average bytes allocated per operation
- `1 allocs/op` - Average number of allocations per operation
- `-8` - GOMAXPROCS value during test

### Performance Grades

| Grade | ns/op Range | Quality |
|-------|-------------|---------|
| A+ | < 1,000 | Excellent |
| A | 1,000-5,000 | Very Good |
| B+ | 5,000-10,000 | Good |
| B | 10,000-50,000 | Acceptable |
| C | 50,000-100,000 | Needs Improvement |
| D | > 100,000 | Poor |

---

## Troubleshooting

### Benchmarks Not Running

**Issue:** `no benchmarks to run`

**Solution:** Ensure you're in the correct directory:
```bash
cd d:\dev\projetos\chain
go test -bench=. -benchmem
```

### Benchmark Fails

**Issue:** Benchmark panics or fails

**Solution:** Check for route conflicts. Each benchmark creates its own router to avoid conflicts.

### Results Vary Widely

**Issue:** Inconsistent results between runs

**Solution:**
1. Close other applications
2. Run multiple times: `go test -bench=. -benchmem -count=5`
3. Use `benchstat` to analyze variance

---

## Documentation

For detailed benchmark results and analysis, see:
- `docs/06-benchmark-results.md` - Complete performance analysis
- `docs/BENCHMARK_PHASE2_2_SUMMARY.md` - Implementation summary

---

## Contributing

When adding new benchmarks:
1. Follow naming convention: `BenchmarkCategory_Description`
2. Always call `b.ReportAllocs()` and `b.ResetTimer()`
3. Create isolated router for each benchmark
4. Add to documentation

---

*Last Updated: April 18, 2026*
