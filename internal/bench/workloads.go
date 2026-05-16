package bench

import (
	"context"
	"math"
	"math/bits"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	defaultBlockSize = 64 * 1024
	kib              = 1024
	mib              = 1024 * kib
)

var sinkUint64 uint64

func deadlineFrom(config WorkerConfig) time.Time {
	duration := config.Duration
	if duration <= 0 {
		duration = time.Second
	}
	return time.Now().Add(duration)
}

func workloadScale(config WorkerConfig, fallback int) int {
	if config.BlockSize > 0 {
		return config.BlockSize
	}
	return fallback
}

func shouldContinue(ctx context.Context, deadline time.Time) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return time.Now().Before(deadline)
	}
}

func fillBytes(size int, seed uint64) []byte {
	data := make([]byte, size)
	x := seed
	for i := range data {
		x = splitmix64(x)
		data[i] = byte(x >> 56)
	}
	return data
}

func splitmix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	z := x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func rotl32(v uint32, n int) uint32 {
	return bits.RotateLeft32(v, n)
}

func runIntegerCrypto(ctx context.Context, config WorkerConfig) WorkerResult {
	const blockBytes = 64
	deadline := deadlineFrom(config)
	var ops uint64
	var bytes uint64
	s0 := uint32(0x61707865 ^ config.WorkerID)
	s1 := uint32(0x3320646e)
	s2 := uint32(0x79622d32)
	s3 := uint32(0x6b206574)

	for shouldContinue(ctx, deadline) {
		for i := 0; i < 256; i++ {
			s0 += s1
			s3 ^= s0
			s3 = rotl32(s3, 16)
			s2 += s3
			s1 ^= s2
			s1 = rotl32(s1, 12)
			s0 += s1
			s3 ^= s0
			s3 = rotl32(s3, 8)
			s2 += s3
			s1 ^= s2
			s1 = rotl32(s1, 7)
		}
		ops += 256 * 8
		bytes += 256 * blockBytes
	}
	atomic.AddUint64(&sinkUint64, uint64(s0)^uint64(s1)^uint64(s2)^uint64(s3))
	return WorkerResult{Operations: ops, Bytes: bytes}
}

func runHash(ctx context.Context, config WorkerConfig) WorkerResult {
	size := config.BlockSize
	if size <= 0 {
		size = defaultBlockSize
	}
	data := fillBytes(size, 0x1234+uint64(config.WorkerID))
	deadline := deadlineFrom(config)
	var bytes uint64
	var ops uint64
	hash := uint64(0xcbf29ce484222325)

	for shouldContinue(ctx, deadline) {
		for _, b := range data {
			hash ^= uint64(b)
			hash *= 0x100000001b3
			hash ^= hash >> 32
			hash = bits.RotateLeft64(hash, 13)
		}
		bytes += uint64(len(data))
		ops += uint64(len(data)) * 4
	}
	atomic.AddUint64(&sinkUint64, hash)
	return WorkerResult{Operations: ops, Bytes: bytes}
}

func runCompression(ctx context.Context, config WorkerConfig) WorkerResult {
	size := config.BlockSize
	if size <= 0 {
		size = defaultBlockSize
	}
	data := make([]byte, size)
	for i := range data {
		if i%97 < 64 {
			data[i] = byte((i / 3) % 251)
		} else {
			data[i] = byte(splitmix64(uint64(i)+uint64(config.WorkerID)) >> 56)
		}
	}

	table := make([]int, 1<<16)
	for i := range table {
		table[i] = -1
	}
	deadline := deadlineFrom(config)
	var bytes uint64
	var ops uint64
	var matches uint64

	for shouldContinue(ctx, deadline) {
		for i := range table {
			table[i] = -1
		}
		for i := 0; i+4 < len(data); i++ {
			key := uint16(data[i])<<8 | uint16(data[i+1])
			prev := table[key]
			if prev >= 0 && i-prev <= 65535 && data[prev+2] == data[i+2] && data[prev+3] == data[i+3] {
				matches++
				i += 3
			}
			table[key] = i
			ops += 6
		}
		bytes += uint64(len(data))
	}
	atomic.AddUint64(&sinkUint64, matches)
	return WorkerResult{Operations: ops, Bytes: bytes, Extra: map[string]float64{"matches": float64(matches)}}
}

func runFloatScience(ctx context.Context, config WorkerConfig) WorkerResult {
	deadline := deadlineFrom(config)
	var ops uint64
	x := 0.125 + float64(config.WorkerID)*0.001
	acc := 1.0
	for shouldContinue(ctx, deadline) {
		for i := 0; i < 512; i++ {
			x += 0.000001
			acc += math.Sin(x)*math.Cos(acc*0.0001) + math.Sqrt(x+1.0)
			acc -= math.Log1p(math.Abs(math.Sin(acc)) + 0.001)
		}
		ops += 512 * 6
	}
	atomic.AddUint64(&sinkUint64, math.Float64bits(acc))
	return WorkerResult{Operations: ops}
}

type vec3 struct {
	x float64
	y float64
	z float64
}

func runVector3D(ctx context.Context, config WorkerConfig) WorkerResult {
	const n = 4096
	a := make([]vec3, n)
	b := make([]vec3, n)
	for i := 0; i < n; i++ {
		a[i] = vec3{x: float64(i%17) + 0.5, y: float64(i%31) + 1.5, z: float64(i%47) + 2.5}
		b[i] = vec3{x: float64(i%23) + 3.5, y: float64(i%29) + 4.5, z: float64(i%37) + 5.5}
	}
	deadline := deadlineFrom(config)
	var ops uint64
	acc := 0.0
	for shouldContinue(ctx, deadline) {
		for i := 0; i < n; i++ {
			cx := a[i].y*b[i].z - a[i].z*b[i].y
			cy := a[i].z*b[i].x - a[i].x*b[i].z
			cz := a[i].x*b[i].y - a[i].y*b[i].x
			dot := a[i].x*b[i].x + a[i].y*b[i].y + a[i].z*b[i].z
			inv := 1.0 / math.Sqrt(cx*cx+cy*cy+cz*cz+dot*dot+1.0)
			acc += (cx + cy + cz + dot) * inv
		}
		ops += n * 24
	}
	atomic.AddUint64(&sinkUint64, math.Float64bits(acc))
	return WorkerResult{Operations: ops}
}

func runFFT(ctx context.Context, config WorkerConfig) WorkerResult {
	const n = 1024
	realPart := make([]float64, n)
	imagPart := make([]float64, n)
	for i := 0; i < n; i++ {
		realPart[i] = math.Sin(float64(i) * 0.01)
		imagPart[i] = math.Cos(float64(i) * 0.013)
	}
	deadline := deadlineFrom(config)
	var transforms uint64
	var flops uint64
	for shouldContinue(ctx, deadline) {
		fftInPlace(realPart, imagPart)
		transforms++
		flops += uint64(5 * n * 10)
	}
	atomic.AddUint64(&sinkUint64, math.Float64bits(realPart[0]+imagPart[1]))
	return WorkerResult{Operations: transforms, Extra: map[string]float64{"estimated_flops": float64(flops)}}
}

func fftInPlace(realPart, imagPart []float64) {
	n := len(realPart)
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j &^= bit
		}
		j |= bit
		if i < j {
			realPart[i], realPart[j] = realPart[j], realPart[i]
			imagPart[i], imagPart[j] = imagPart[j], imagPart[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		wlenR := math.Cos(angle)
		wlenI := math.Sin(angle)
		for i := 0; i < n; i += length {
			wr, wi := 1.0, 0.0
			for k := 0; k < length/2; k++ {
				uR := realPart[i+k]
				uI := imagPart[i+k]
				vR := realPart[i+k+length/2]*wr - imagPart[i+k+length/2]*wi
				vI := realPart[i+k+length/2]*wi + imagPart[i+k+length/2]*wr
				realPart[i+k] = uR + vR
				imagPart[i+k] = uI + vI
				realPart[i+k+length/2] = uR - vR
				imagPart[i+k+length/2] = uI - vI
				nextWR := wr*wlenR - wi*wlenI
				wi = wr*wlenI + wi*wlenR
				wr = nextWR
			}
		}
	}
}

func runRender(ctx context.Context, config WorkerConfig) WorkerResult {
	deadline := deadlineFrom(config)
	var pixels uint64
	var checksum uint64
	yOffset := config.WorkerID * 97
	for shouldContinue(ctx, deadline) {
		for y := 0; y < 96; y++ {
			cy := (float64(y+yOffset)/96.0)*2.0 - 1.0
			for x := 0; x < 128; x++ {
				cx := (float64(x)/128.0)*3.5 - 2.5
				zx, zy := 0.0, 0.0
				iter := 0
				for ; iter < 32 && zx*zx+zy*zy < 4.0; iter++ {
					xt := zx*zx - zy*zy + cx
					zy = 2*zx*zy + cy
					zx = xt
				}
				checksum += uint64(iter)
			}
		}
		pixels += 96 * 128
	}
	atomic.AddUint64(&sinkUint64, checksum)
	return WorkerResult{Operations: pixels}
}

func runCompileModel(ctx context.Context, config WorkerConfig) WorkerResult {
	source := generateSource(16*kib, uint64(config.WorkerID)+1)
	deadline := deadlineFrom(config)
	var files uint64
	var tokens uint64
	for shouldContinue(ctx, deadline) {
		tokens += uint64(scanAndParse(source))
		files++
	}
	atomic.AddUint64(&sinkUint64, tokens)
	return WorkerResult{Operations: files, Bytes: files * uint64(len(source)), Extra: map[string]float64{"tokens": float64(tokens)}}
}

func generateSource(size int, seed uint64) []byte {
	words := []string{"func", "var", "return", "if", "else", "for", "value", "index", "total", "package"}
	out := make([]byte, 0, size)
	x := seed
	for len(out) < size {
		x = splitmix64(x)
		word := words[int(x%uint64(len(words)))]
		out = append(out, word...)
		switch x % 7 {
		case 0:
			out = append(out, '(', ')', ' ', '{', '\n')
		case 1:
			out = append(out, ' ', '=', ' ', byte('0'+x%10), '\n')
		case 2:
			out = append(out, ';', '\n')
		default:
			out = append(out, ' ')
		}
	}
	return out[:size]
}

func scanAndParse(source []byte) int {
	tokens := 0
	state := 0
	hash := uint64(1469598103934665603)
	for _, b := range source {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' {
			hash ^= uint64(b)
			hash *= 1099511628211
			state = (state + int(b)) & 31
			continue
		}
		if state != 0 {
			tokens++
			state = 0
		}
		switch b {
		case '{', '}', '(', ')', '=', ';':
			tokens++
			hash = bits.RotateLeft64(hash, 5) ^ uint64(b)
		}
	}
	atomic.AddUint64(&sinkUint64, hash)
	return tokens
}

func runBatchData(ctx context.Context, config WorkerConfig) WorkerResult {
	const n = 8192
	data := make([]uint64, n)
	for i := range data {
		data[i] = splitmix64(uint64(i) + uint64(config.WorkerID)*0x9e37)
	}
	deadline := deadlineFrom(config)
	var rows uint64
	var acc uint64
	for shouldContinue(ctx, deadline) {
		for i := range data {
			v := data[i]
			if v&7 != 0 {
				v = bits.RotateLeft64(v^0xa0761d6478bd642f, int(v&31))
				acc += v % 1009
			}
			data[i] = v + uint64(i)
		}
		rows += n
	}
	atomic.AddUint64(&sinkUint64, acc)
	return WorkerResult{Operations: rows}
}

func runMemoryStream(ctx context.Context, config WorkerConfig) WorkerResult {
	size := workloadScale(config, 4*mib)
	if size < 8*kib {
		size = 8 * kib
	}
	a := make([]float64, size/8)
	b := make([]float64, size/8)
	c := make([]float64, size/8)
	for i := range a {
		a[i] = float64(i%251) + 1.0
		b[i] = float64(i%127) + 2.0
	}
	deadline := deadlineFrom(config)
	var bytes uint64
	sum := 0.0
	for shouldContinue(ctx, deadline) {
		for i := range a {
			c[i] = a[i]
		}
		for i := range a {
			b[i] = 3.0 * c[i]
		}
		for i := range a {
			c[i] = a[i] + b[i]
		}
		for i := range a {
			a[i] = b[i] + 0.5*c[i]
			sum += a[i]
		}
		bytes += uint64(len(a)) * 8 * (2 + 2 + 3 + 3)
	}
	atomic.AddUint64(&sinkUint64, math.Float64bits(sum))
	return WorkerResult{Bytes: bytes}
}

func runPointerChase(ctx context.Context, config WorkerConfig) WorkerResult {
	n := workloadScale(config, 1<<20)
	if n < 1024 {
		n = 1024
	}
	n = 1 << bits.Len(uint(n)-1)
	next := make([]uint32, n)
	order := make([]uint32, n)
	for i := range order {
		order[i] = uint32(i)
	}
	x := uint64(0xfeedface) + uint64(config.WorkerID)
	for i := n - 1; i > 0; i-- {
		x = splitmix64(x)
		j := int(x % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}
	for i := 0; i < n; i++ {
		next[order[i]] = order[(i+1)%n]
	}

	deadline := deadlineFrom(config)
	var steps uint64
	index := order[0]
	for shouldContinue(ctx, deadline) {
		for i := 0; i < 4096; i++ {
			index = next[index]
		}
		steps += 4096
	}
	atomic.AddUint64(&sinkUint64, uint64(index))
	return WorkerResult{Operations: steps, Bytes: steps * 4}
}

func runCacheHierarchy(ctx context.Context, config WorkerConfig) WorkerResult {
	largest := workloadScale(config, 4*mib)
	if largest < 256*kib {
		largest = 256 * kib
	}
	sizes := []int{32 * kib, 256 * kib, largest}
	sets := make([][]uint64, len(sizes))
	for idx, size := range sizes {
		sets[idx] = make([]uint64, size/8)
		for i := range sets[idx] {
			sets[idx][i] = uint64(i)
		}
	}
	deadline := deadlineFrom(config)
	var bytes uint64
	var ops uint64
	acc := uint64(0)
	extra := map[string]float64{}
	for shouldContinue(ctx, deadline) {
		for idx, size := range sizes {
			data := sets[idx]
			before := time.Now()
			localBytes := uint64(0)
			for repeat := 0; repeat < 32; repeat++ {
				for i := 0; i < len(data); i += 8 {
					acc += data[i]
				}
				localBytes += uint64(len(data)/8) * 8
			}
			elapsed := time.Since(before).Seconds()
			if elapsed > 0 {
				extra[cacheLabel(size)+"_gib_s"] += float64(localBytes) / elapsed / (1024 * 1024 * 1024)
				extra[cacheLabel(size)+"_samples"]++
			}
			bytes += localBytes
			ops += uint64(len(data) / 8 * 32)
		}
	}
	for _, size := range sizes {
		label := cacheLabel(size)
		if samples := extra[label+"_samples"]; samples > 0 {
			extra[label+"_gib_s"] /= samples
		}
		delete(extra, label+"_samples")
	}
	atomic.AddUint64(&sinkUint64, acc)
	return WorkerResult{Operations: ops, Bytes: bytes, Extra: extra}
}

func cacheLabel(size int) string {
	switch size {
	case 32 * kib:
		return "l1_like"
	case 256 * kib:
		return "l2_like"
	default:
		return "l3_like"
	}
}

func runCoherency(ctx context.Context, config WorkerConfig) WorkerResult {
	deadline := deadlineFrom(config)
	var local uint64
	for shouldContinue(ctx, deadline) {
		for i := 0; i < 1024; i++ {
			atomic.AddUint64(&sinkUint64, 1)
		}
		local += 1024
		runtime.Gosched()
	}
	return WorkerResult{Operations: local}
}

func runSIMDBaseline(ctx context.Context, config WorkerConfig) WorkerResult {
	const n = 8192
	a := make([]float64, n)
	b := make([]float64, n)
	c := make([]float64, n)
	for i := range a {
		a[i] = float64(i%113) * 0.5
		b[i] = float64(i%97) * 0.25
		c[i] = float64(i%53) * 0.125
	}
	deadline := deadlineFrom(config)
	var flops uint64
	sum := 0.0
	for shouldContinue(ctx, deadline) {
		for i := range a {
			c[i] = a[i]*b[i] + c[i]
			a[i] = c[i]*0.999999 + b[i]
			sum += a[i]
		}
		flops += n * 4
	}
	atomic.AddUint64(&sinkUint64, math.Float64bits(sum))
	return WorkerResult{Operations: flops, Bytes: flops / 4 * 8 * 3}
}

func runBranch(ctx context.Context, config WorkerConfig) WorkerResult {
	const n = 16384
	data := make([]uint64, n)
	x := uint64(0x123456789abcdef0) + uint64(config.WorkerID)
	for i := range data {
		x = splitmix64(x)
		data[i] = x
	}
	deadline := deadlineFrom(config)
	var branches uint64
	acc := uint64(0)
	for shouldContinue(ctx, deadline) {
		for _, v := range data {
			if v&1 == 0 {
				acc += v ^ 0x9e3779b97f4a7c15
			} else if v&2 == 0 {
				acc ^= bits.RotateLeft64(v, 17)
			} else if v&4 == 0 {
				acc -= v | 1
			} else {
				acc += v * 3
			}
			if acc&0xff == 17 {
				acc = bits.RotateLeft64(acc, 3)
			}
		}
		branches += n * 5
	}
	atomic.AddUint64(&sinkUint64, acc)
	return WorkerResult{Operations: branches}
}

func runDependencyChain(ctx context.Context, config WorkerConfig) WorkerResult {
	deadline := deadlineFrom(config)
	var ops uint64
	x := uint64(0x9e3779b97f4a7c15) + uint64(config.WorkerID)
	for shouldContinue(ctx, deadline) {
		for i := 0; i < 4096; i++ {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
			x *= 0xbf58476d1ce4e5b9
		}
		ops += 4096 * 4
	}
	atomic.AddUint64(&sinkUint64, x)
	return WorkerResult{Operations: ops}
}

func runEfficiencyProxy(ctx context.Context, config WorkerConfig) WorkerResult {
	startCPU := time.Now()
	deadline := deadlineFrom(config)
	var score uint64
	x := uint64(0x243f6a8885a308d3) + uint64(config.WorkerID)
	y := 0.5 + float64(config.WorkerID)
	for shouldContinue(ctx, deadline) {
		for i := 0; i < 2048; i++ {
			x = splitmix64(x)
			y += math.Sqrt(float64(x&0xffff)+y) * 0.00001
			if x&3 == 0 {
				y *= 0.999999
			}
		}
		score += 2048
	}
	atomic.AddUint64(&sinkUint64, x^math.Float64bits(y))
	return WorkerResult{
		Operations: score,
		Extra: map[string]float64{
			"cpu_seconds": time.Since(startCPU).Seconds(),
			"note":        0,
		},
	}
}
