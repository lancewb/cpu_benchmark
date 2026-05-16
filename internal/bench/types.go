package bench

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"
)

type WorkloadFunc func(context.Context, WorkerConfig) WorkerResult

type Benchmark struct {
	ID          string
	Name        string
	Dimension   string
	Focus       string
	Scenario    string
	Unit        string
	Parallel    bool
	Description string
	Run         WorkloadFunc
}

type WorkerConfig struct {
	Duration  time.Duration
	Workers   int
	WorkerID  int
	BlockSize int
}

type WorkerResult struct {
	Operations uint64
	Bytes      uint64
	Extra      map[string]float64
}

type Result struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Dimension      string             `json:"dimension"`
	Focus          string             `json:"focus"`
	Scenario       string             `json:"scenario"`
	Duration       float64            `json:"duration_seconds"`
	Workers        int                `json:"workers"`
	Operations     uint64             `json:"operations"`
	Bytes          uint64             `json:"bytes"`
	OpsPerSecond   float64            `json:"ops_per_second"`
	BytesPerSecond float64            `json:"bytes_per_second"`
	Score          float64            `json:"score"`
	Unit           string             `json:"unit"`
	Extra          map[string]float64 `json:"extra,omitempty"`
}

type RunOptions struct {
	IDs      []string
	Duration time.Duration
	Workers  int
}

func NormalizeOptions(options RunOptions) RunOptions {
	if options.Duration <= 0 {
		options.Duration = time.Second
	}
	if options.Workers <= 0 {
		options.Workers = runtime.NumCPU()
	}
	return options
}

func Catalog() []Benchmark {
	items := []Benchmark{
		{
			ID:          "int-crypto",
			Name:        "Integer crypto stream",
			Dimension:   "single-core integer",
			Focus:       "纯软件 ARX 加密风格整数运算、位移、异或、加法和状态混合",
			Scenario:    "TLS/加密库、游戏逻辑、压缩前处理等单线程整数热点",
			Unit:        "MiB/s",
			Parallel:    false,
			Description: "Measures scalar integer throughput with a ChaCha-like round function.",
			Run:         runIntegerCrypto,
		},
		{
			ID:          "hash",
			Name:        "Hash mixing",
			Dimension:   "single-core integer",
			Focus:       "64 位哈希混合、乘法、旋转和内存顺序读取",
			Scenario:    "哈希表、去重、日志处理、数据库 key 查找",
			Unit:        "MiB/s",
			Parallel:    false,
			Description: "Measures non-cryptographic hash style integer mixing throughput.",
			Run:         runHash,
		},
		{
			ID:          "compress",
			Name:        "Compression model",
			Dimension:   "single-core integer",
			Focus:       "LZ 类匹配查找、字节扫描、短距离重复数据检测",
			Scenario:    "归档、日志压缩、网络数据压缩的单线程吞吐",
			Unit:        "MiB/s",
			Parallel:    false,
			Description: "Portable LZ-style match finding workload; it is not a real codec.",
			Run:         runCompression,
		},
		{
			ID:          "float-sci",
			Name:        "Scientific floating point",
			Dimension:   "single-core floating point",
			Focus:       "sin/cos/sqrt/log 等标量浮点数学函数与循环依赖",
			Scenario:    "CAD、仿真、统计计算、几何求解",
			Unit:        "Mops/s",
			Parallel:    false,
			Description: "Measures scalar floating-point math throughput using standard library functions.",
			Run:         runFloatScience,
		},
		{
			ID:          "vec3d",
			Name:        "3D vector math",
			Dimension:   "single-core floating point",
			Focus:       "点积、叉积、归一化、矩阵风格累加",
			Scenario:    "3D 引擎、物理模拟、CAD 视图变换",
			Unit:        "Mops/s",
			Parallel:    false,
			Description: "Measures portable 3D vector arithmetic and normalization.",
			Run:         runVector3D,
		},
		{
			ID:          "fft",
			Name:        "FFT transform",
			Dimension:   "single-core floating point",
			Focus:       "复数蝶形运算、三角函数预计算、缓存友好顺序访问",
			Scenario:    "音频处理、信号分析、频域滤波",
			Unit:        "transforms/s",
			Parallel:    false,
			Description: "Measures iterative radix-2 FFT throughput on a fixed-size buffer.",
			Run:         runFFT,
		},
		{
			ID:          "render",
			Name:        "Parallel tile render",
			Dimension:   "multi-core parallel",
			Focus:       "多线程像素块渲染、浮点与整数混合、线程调度",
			Scenario:    "离线渲染、图像处理、视频滤镜",
			Unit:        "Mpixels/s",
			Parallel:    true,
			Description: "Measures multi-worker tile rendering throughput with a Mandelbrot-style kernel.",
			Run:         runRender,
		},
		{
			ID:          "compile",
			Name:        "Parallel compile model",
			Dimension:   "multi-core parallel",
			Focus:       "词法扫描、解析状态机、符号表哈希和批量小任务",
			Scenario:    "并行编译、静态分析、代码生成",
			Unit:        "files/s",
			Parallel:    true,
			Description: "Measures a compiler-like workload without invoking an external compiler.",
			Run:         runCompileModel,
		},
		{
			ID:          "batch",
			Name:        "Batch data processing",
			Dimension:   "multi-core parallel",
			Focus:       "批量过滤、聚合、排序片段、数据转换",
			Scenario:    "ETL、服务端批处理、指标计算",
			Unit:        "Mrows/s",
			Parallel:    true,
			Description: "Measures parallel data transformation and aggregation throughput.",
			Run:         runBatchData,
		},
		{
			ID:          "mem-stream",
			Name:        "Memory stream bandwidth",
			Dimension:   "memory bandwidth and latency",
			Focus:       "顺序读写、copy/scale/add/triad 风格内存带宽",
			Scenario:    "大数组计算、媒体处理、数据库扫描",
			Unit:        "GiB/s",
			Parallel:    true,
			Description: "Measures portable STREAM-style sequential memory bandwidth.",
			Run:         runMemoryStream,
		},
		{
			ID:          "mem-latency",
			Name:        "Pointer chasing latency",
			Dimension:   "memory bandwidth and latency",
			Focus:       "随机指针追踪、缓存/TLB 未命中、依赖加载延迟",
			Scenario:    "图算法、索引查找、低局部性服务端负载",
			Unit:        "Msteps/s",
			Parallel:    false,
			Description: "Measures dependent random memory access latency using a shuffled pointer ring.",
			Run:         runPointerChase,
		},
		{
			ID:          "cache",
			Name:        "Cache hierarchy",
			Dimension:   "cache subsystem",
			Focus:       "不同工作集大小下的缓存带宽与局部性退化",
			Scenario:    "数据密集型循环、热数据结构设计、多级缓存敏感代码",
			Unit:        "GiB/s",
			Parallel:    false,
			Description: "Measures repeated access to L1/L2/L3-sized working sets.",
			Run:         runCacheHierarchy,
		},
		{
			ID:          "coherency",
			Name:        "Cross-core coherency",
			Dimension:   "cache subsystem",
			Focus:       "跨线程原子计数、缓存行争用、缓存一致性通信",
			Scenario:    "锁竞争、共享计数器、队列调度、多核扩展性瓶颈",
			Unit:        "Mops/s",
			Parallel:    true,
			Description: "Measures shared atomic increment throughput as a coherency stress test.",
			Run:         runCoherency,
		},
		{
			ID:          "simd",
			Name:        "Portable vector baseline",
			Dimension:   "SIMD/vector instruction",
			Focus:       "连续数组向量化友好的 FMA 风格循环，由编译器在各架构上自行优化",
			Scenario:    "多媒体、AI 推理、密码学中可向量化的热点循环",
			Unit:        "GFLOP/s",
			Parallel:    false,
			Description: "Measures compiler-vectorizable scalar code; it does not force SSE/AVX/NEON.",
			Run:         runSIMDBaseline,
		},
		{
			ID:          "branch",
			Name:        "Branch prediction",
			Dimension:   "branch prediction and OoO",
			Focus:       "高分支密度、可预测/不可预测分支混合、条件数据依赖",
			Scenario:    "解释器、规则引擎、压缩/解析、游戏 AI",
			Unit:        "Mbranches/s",
			Parallel:    false,
			Description: "Measures throughput of branch-heavy code with deterministic pseudo-random input.",
			Run:         runBranch,
		},
		{
			ID:          "dependency",
			Name:        "Dependency chain",
			Dimension:   "branch prediction and OoO",
			Focus:       "长依赖链、乱序执行隐藏延迟能力、整数流水线延迟",
			Scenario:    "串行状态机、密码学反馈模式、低并行度算法",
			Unit:        "Mops/s",
			Parallel:    false,
			Description: "Measures serial integer dependency chain throughput.",
			Run:         runDependencyChain,
		},
		{
			ID:          "efficiency",
			Name:        "Performance efficiency proxy",
			Dimension:   "power efficiency",
			Focus:       "固定时间内综合吞吐与 CPU 时间消耗的比值；不读取硬件功耗",
			Scenario:    "无功耗传感器环境下粗略比较调度效率和热降频趋势",
			Unit:        "score/cpu-second",
			Parallel:    true,
			Description: "Reports a performance-per-CPU-time proxy, not true watts.",
			Run:         runEfficiencyProxy,
		},
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func Find(id string) (Benchmark, bool) {
	for _, item := range Catalog() {
		if item.ID == id {
			return item, true
		}
	}
	return Benchmark{}, false
}

func ParseIDs(content string) ([]string, error) {
	content = strings.TrimSpace(content)
	if content == "" || strings.EqualFold(content, "all") {
		return allIDs(), nil
	}

	seen := map[string]bool{}
	ids := []string{}
	for _, raw := range strings.Split(content, ",") {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := Find(id); !ok {
			return nil, fmt.Errorf("unknown benchmark content %q", id)
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no benchmark content selected")
	}
	return ids, nil
}

func Run(ctx context.Context, options RunOptions) ([]Result, error) {
	options = NormalizeOptions(options)
	if len(options.IDs) == 0 {
		options.IDs = allIDs()
	}

	results := make([]Result, 0, len(options.IDs))
	for _, id := range options.IDs {
		item, ok := Find(id)
		if !ok {
			return nil, fmt.Errorf("unknown benchmark content %q", id)
		}
		workers := 1
		if item.Parallel {
			workers = options.Workers
		}
		start := time.Now()
		combined := runWorkers(ctx, item.Run, WorkerConfig{
			Duration:  options.Duration,
			Workers:   workers,
			BlockSize: defaultBlockSize,
		}, workers)
		elapsed := time.Since(start)
		results = append(results, buildResult(item, combined, elapsed, workers))
		if err := ctx.Err(); err != nil {
			return results, err
		}
	}
	return results, nil
}

func allIDs() []string {
	items := Catalog()
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func runWorkers(ctx context.Context, workload WorkloadFunc, config WorkerConfig, workers int) WorkerResult {
	if workers <= 1 {
		config.Workers = 1
		config.WorkerID = 0
		return workload(ctx, config)
	}

	ch := make(chan WorkerResult, workers)
	for id := 0; id < workers; id++ {
		workerConfig := config
		workerConfig.WorkerID = id
		workerConfig.Workers = workers
		go func() {
			ch <- workload(ctx, workerConfig)
		}()
	}

	var total WorkerResult
	for i := 0; i < workers; i++ {
		total = mergeResult(total, <-ch)
	}
	return total
}

func mergeResult(left, right WorkerResult) WorkerResult {
	left.Operations += right.Operations
	left.Bytes += right.Bytes
	if len(right.Extra) > 0 {
		if left.Extra == nil {
			left.Extra = map[string]float64{}
		}
		for key, value := range right.Extra {
			left.Extra[key] += value
		}
	}
	return left
}

func buildResult(item Benchmark, raw WorkerResult, elapsed time.Duration, workers int) Result {
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		seconds = 1e-9
	}
	score := scoreFor(item.Unit, raw, seconds)
	return Result{
		ID:             item.ID,
		Name:           item.Name,
		Dimension:      item.Dimension,
		Focus:          item.Focus,
		Scenario:       item.Scenario,
		Duration:       seconds,
		Workers:        workers,
		Operations:     raw.Operations,
		Bytes:          raw.Bytes,
		OpsPerSecond:   float64(raw.Operations) / seconds,
		BytesPerSecond: float64(raw.Bytes) / seconds,
		Score:          score,
		Unit:           item.Unit,
		Extra:          raw.Extra,
	}
}

func scoreFor(unit string, raw WorkerResult, seconds float64) float64 {
	switch unit {
	case "MiB/s":
		return float64(raw.Bytes) / seconds / (1024 * 1024)
	case "GiB/s":
		return float64(raw.Bytes) / seconds / (1024 * 1024 * 1024)
	case "Mops/s", "Msteps/s", "Mbranches/s":
		return float64(raw.Operations) / seconds / 1_000_000
	case "GFLOP/s":
		return float64(raw.Operations) / seconds / 1_000_000_000
	case "Mpixels/s", "Mrows/s":
		return float64(raw.Operations) / seconds / 1_000_000
	case "transforms/s", "files/s":
		return float64(raw.Operations) / seconds
	case "score/cpu-second":
		if cpuSeconds := raw.Extra["cpu_seconds"]; cpuSeconds > 0 {
			return float64(raw.Operations) / cpuSeconds
		}
		return float64(raw.Operations) / seconds
	default:
		return float64(raw.Operations) / seconds
	}
}
