// compression-bench measures candidate replacements for the node-status
// compression in workflow/packer (JSON+gzip+base64). It synthesizes node
// statuses from the examples/ specs, trains zstd dictionaries on half the
// corpus, and reports sizes/ratios/timings for each codec on the held-out
// half. See docs/superpowers/specs/2026-06-12-node-compression-benchmark-design.md.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/klauspost/compress/dict"
	"github.com/klauspost/compress/zstd"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func main() {
	examplesDir := flag.String("examples", "examples", "directory of example workflow specs")
	scalesFlag := flag.String("scales", "100,1000,5000,10000", "comma-separated target node counts")
	specsPerScale := flag.Int("specs-per-scale", 16, "workflows synthesized per scale (half train the dictionary, half are measured)")
	seed := flag.Int64("seed", 42, "rng seed for corpus synthesis")
	dictSize := flag.Int("dict-size", 112640, "max dictionary size in bytes (zstd default 110KiB)")
	zstdLevel := flag.Int("zstd-level", 3, "zstd encoder level: 1=fastest 2=default 3=better 4=best")
	brotliLevels := flag.String("brotli-levels", "", "comma-separated brotli qualities to include, e.g. 5,7,9,11 (empty: none)")
	flag.Parse()

	if err := run(*examplesDir, *scalesFlag, *specsPerScale, *seed, *dictSize, *zstdLevel, *brotliLevels); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(examplesDir, scalesFlag string, specsPerScale int, seed int64, dictSize, zstdLevel int, brotliLevelsFlag string) error {
	// Error-level logger: SplitWorkflowYAMLFile and util/file log through the
	// context, and per-file parse noise isn't interesting here.
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.Error, logging.Text))

	var scales []int
	for _, s := range strings.Split(scalesFlag, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 2 {
			return fmt.Errorf("bad scale %q", s)
		}
		scales = append(scales, n)
	}
	if specsPerScale < 2 {
		return fmt.Errorf("-specs-per-scale must be >= 2 (half train, half eval)")
	}
	level := zstd.EncoderLevel(zstdLevel)
	if level < zstd.SpeedFastest || level > zstd.SpeedBestCompression {
		return fmt.Errorf("-zstd-level must be 1..4")
	}
	var brotliLevels []int
	if brotliLevelsFlag != "" {
		for _, s := range strings.Split(brotliLevelsFlag, ",") {
			q, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || q < 0 || q > 11 {
				return fmt.Errorf("bad brotli quality %q (must be 0..11)", s)
			}
			brotliLevels = append(brotliLevels, q)
		}
	}

	specs, err := loadSpecs(ctx, examplesDir)
	if err != nil {
		return err
	}
	fmt.Printf("loaded %d workflow specs from %s\n", len(specs), examplesDir)

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(specs), func(i, j int) { specs[i], specs[j] = specs[j], specs[i] })

	// Synthesize per-scale corpora; the first half of each scale trains the
	// dictionaries, only the second half is measured.
	train := map[int][]wfv1.Nodes{}
	eval := map[int][]wfv1.Nodes{}
	for _, scale := range scales {
		for i := 0; i < specsPerScale; i++ {
			spec := &specs[(scale+i)%len(specs)]
			nodes := synthesizeNodes(spec, scale, rng)
			if i < specsPerScale/2 {
				train[scale] = append(train[scale], nodes)
			} else {
				eval[scale] = append(eval[scale], nodes)
			}
		}
	}

	jsonDict, protoDict, err := trainDicts(train, dictSize, level)
	if err != nil {
		return err
	}
	fmt.Printf("trained dictionaries: json=%dB proto=%dB\n\n", len(jsonDict), len(protoDict))

	codecs, err := buildCodecs(ctx, level, jsonDict, protoDict, brotliLevels)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SCALE\tCODEC\tJSON\tCOMPRESSED\tBASE64\tVS GZIP\tENC ms\tDEC ms")
	for _, scale := range scales {
		var baseline int64
		for _, c := range codecs {
			r, err := measure(c, eval[scale])
			if err != nil {
				return fmt.Errorf("%s @ %d nodes: %w", c.name, scale, err)
			}
			if baseline == 0 {
				baseline = r.b64 // first codec is json+gzip
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%.1f%%\t%.1f\t%.1f\n",
				scale, c.name, size(r.orig), size(r.compressed), size(r.b64),
				100*float64(r.b64)/float64(baseline), r.encMs, r.decMs)
		}
		fmt.Fprintln(w, "\t\t\t\t\t\t\t")
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Println("JSON/COMPRESSED/BASE64 are averages per workflow over the held-out blobs; VS GZIP compares base64 sizes (lower is better).")
	return nil
}

// trainDicts builds one JSON-trained and one proto-trained dictionary from
// all training blobs across every scale, matching deployment reality: a
// single shipped dictionary, whatever the workflow size.
func trainDicts(train map[int][]wfv1.Nodes, maxSize int, level zstd.EncoderLevel) ([]byte, []byte, error) {
	var jsonSamples, protoSamples [][]byte
	for _, blobs := range train {
		for _, nodes := range blobs {
			j, err := marshalJSON(nodes)
			if err != nil {
				return nil, nil, err
			}
			p, err := marshalProto(nodes)
			if err != nil {
				return nil, nil, err
			}
			// The dict builder expects many small samples (like zstd --train)
			// and panics on multi-MB inputs, so chunk the blobs.
			jsonSamples = append(jsonSamples, chunk(j, 64<<10)...)
			protoSamples = append(protoSamples, chunk(p, 64<<10)...)
		}
	}
	opts := dict.Options{MaxDictSize: maxSize, HashBytes: 6, ZstdLevel: level, ZstdDictID: 1}
	jsonDict, err := dict.BuildZstdDict(jsonSamples, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("train json dict: %w", err)
	}
	opts.ZstdDictID = 2
	protoDict, err := dict.BuildZstdDict(protoSamples, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("train proto dict: %w", err)
	}
	return jsonDict, protoDict, nil
}

func chunk(b []byte, size int) [][]byte {
	var out [][]byte
	for len(b) > size {
		out = append(out, b[:size])
		b = b[size:]
	}
	return append(out, b)
}

type result struct {
	orig, compressed, b64 int64 // averages per blob, bytes
	encMs, decMs          float64
}

func measure(c codec, blobs []wfv1.Nodes) (result, error) {
	var r result
	var encTotal, decTotal time.Duration
	for _, nodes := range blobs {
		origJSON, err := json.Marshal(nodes)
		if err != nil {
			return r, err
		}

		start := time.Now()
		compressed, err := c.encode(nodes)
		encTotal += time.Since(start)
		if err != nil {
			return r, err
		}

		start = time.Now()
		decoded, err := c.decode(compressed)
		decTotal += time.Since(start)
		if err != nil {
			return r, err
		}

		// Fidelity gate: a lossy codec must fail the run, not post a good ratio.
		decodedJSON, err := json.Marshal(decoded)
		if err != nil {
			return r, err
		}
		if !bytes.Equal(origJSON, decodedJSON) {
			return r, fmt.Errorf("round-trip mismatch: %dB in, %dB out", len(origJSON), len(decodedJSON))
		}

		r.orig += int64(len(origJSON))
		r.compressed += int64(len(compressed))
		r.b64 += int64(base64.StdEncoding.EncodedLen(len(compressed)))
	}
	n := int64(len(blobs))
	r.orig /= n
	r.compressed /= n
	r.b64 /= n
	r.encMs = float64(encTotal.Microseconds()) / float64(n) / 1000
	r.decMs = float64(decTotal.Microseconds()) / float64(n) / 1000
	return r, nil
}

func size(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2fMiB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKiB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
