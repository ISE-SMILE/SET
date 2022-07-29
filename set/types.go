package set

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/faas-facts/bench/bencher"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type Unit int64

const (
	B   Unit = 1
	kiB      = B * 1024
	MiB      = kiB * 1024
	GiB      = MiB * 1024
	TiB      = GiB * 1024
)

var memoryLevel = map[byte]MemoryTask{
	0: {
		OperatorSize:   100,
		Iteration:      10000,
		RecursionDepth: 20,
	},
	1: {
		OperatorSize:   1000,
		Iteration:      10000,
		RecursionDepth: 20,
	},
	2: {
		OperatorSize:   100,
		Iteration:      100000,
		RecursionDepth: 20,
	},
	3: {
		OperatorSize:   1000,
		Iteration:      1000000,
		RecursionDepth: 20,
	},
	4: {
		OperatorSize:   100,
		Iteration:      100000,
		RecursionDepth: 2000,
	},
	5: {
		OperatorSize:   10000,
		Iteration:      100000,
		RecursionDepth: 2000,
	},
	6: {
		OperatorSize:   10000,
		Iteration:      1000000,
		RecursionDepth: 2000,
	},
}
var ioLevel = map[byte]IOTask{
	//Read only 1000 times, 512B out of 10 5 MB files
	0: {
		Iteration:    1000,
		ReadWrite:    0,
		ChunkSize:    int64(512 * B),
		objectNumber: 10,
		objectSize:   int64(5 * MiB),
	},
	//write only 512B 1000 times
	1: {
		Iteration:    1000,
		ReadWrite:    1,
		ChunkSize:    int64(512 * B),
		objectNumber: 0,
		objectSize:   int64(5 * MiB),
	},
	2: {
		Iteration:    10000,
		ReadWrite:    0.5,
		ChunkSize:    int64(1 * MiB),
		objectNumber: 20,
		objectSize:   int64(100 * MiB),
	},
	3: {
		Iteration:    10000,
		ReadWrite:    0.5,
		ChunkSize:    int64(2 * MiB),
		objectNumber: 20,
		objectSize:   int64(100 * MiB),
	},
	4: {
		Iteration:    100000,
		ReadWrite:    0.7,
		ChunkSize:    int64(20 * MiB),
		objectNumber: 10,
		objectSize:   int64(100 * MiB),
	},
	5: {
		Iteration:    10000,
		ReadWrite:    0.7,
		ChunkSize:    int64(50 * MiB),
		objectNumber: 10,
		objectSize:   int64(100 * MiB),
	},
	6: {
		Iteration:    100,
		ReadWrite:    0.7,
		ChunkSize:    int64(100 * MiB),
		objectNumber: 10,
		objectSize:   int64(100 * MiB),
	},
}
var primeLevel = map[byte]int32{
	0: 1e3,
	1: 1e4,
	2: 1e5,
	3: 1e6,
	4: 1e7,
	5: 1e8,
	6: 1e9,
}
var idleLevel = map[byte]int{
	0: 0,
	1: 2,
	2: 8,
	3: 16,
	4: 32,
	5: 64,
	6: 128,
}

type PerformanceWorkload struct {
	//Meta-Data
	Name   string `json:"name" yaml:"name"`
	Target string `json:"target" yaml:"target"`

	//Client-Side Options
	Threads int `json:"threads" yaml:"threads"`

	//Workload Profile
	Warmup      int           `json:"warmup" yaml:"warmup"`
	Scaling     float64       `json:"scaling" yaml:"scaling"`
	PhaseLength time.Duration `json:"phaseLength" yaml:"phaseLength"`
	Type        string        `json:"type" yaml:"type"`
	Level       byte          `json:"complexity" yaml:"complexity"`

	//We trigger this change during the scaleing phase
	Operation *Deployment `json:"opTask" yaml:"opTask"`

	//IO Extras
	Bucket string   `json:"bucket,omitempty" yaml:"bucket"`
	Keys   []string `json:"keys,omitempty" yaml:"keys"`

	Endpoint        string `json:"endpoint,omitempty" yaml:"endpoint"`
	AccessKeyID     string `json:"key_id,omitempty" yaml:"keyId"`
	AccessKeySecret string `json:"key,omitempty" yaml:"secret"`

	DisableSSL  bool   `json:"disableSSL,omitempty" yaml:"S3disableSSL"`
	S3PathStyle bool   `json:"S3PathStyle,omitempty" yaml:"S3PathStyle"`
	S3Region    string `json:"region,omitempty" yaml:"S3Region"`

	//Invoker
	Invoker bencher.InvokerConfig `json:"invoker,omitempty" yaml:"invoker"`

	//Deployment
	Deployment Deployment `json:"deployment,omitempty" yaml:"deployment"`

	Platform Platform
}

func (w *PerformanceWorkload) Prepare() *bencher.Bencher {
	//check if keys is set and generate files otherwise...
	config := bencher.BenchmarkConfig{
		OutputFile: "data/$name_$date.csv",
		Workload: bencher.WorkloadConfig{
			Name:   w.Name,
			Target: w.Target,
			Phases: []bencher.PhaseConfig{
				{
					Name:    "warmup",
					Threads: w.Threads,
					HatchRate: bencher.HatchRateConfig{
						Type: "fixed",
						Options: map[string]interface{}{
							"trps": w.Warmup,
						},
					},
					Timeout: w.PhaseLength,
				},
				{
					Name:    "scale",
					Threads: w.Threads,
					HatchRate: bencher.HatchRateConfig{
						Type: "slope",
						Options: map[string]interface{}{
							"start": w.Warmup,
							"rate":  w.Scaling,
						},
					},
					Timeout: w.PhaseLength,
				},
				{
					Name:    "settle",
					Threads: w.Threads,
					HatchRate: bencher.HatchRateConfig{
						Type: "fixed",
						Options: map[string]interface{}{
							"trps": int(math.Ceil(float64(w.Warmup) + w.PhaseLength.Seconds()*w.Scaling)),
						},
					},
					Timeout: w.PhaseLength,
				},
			},
			Invocation: w.Invoker,
		},
	}

	runner, err := bencher.BencherReadFromConfig(config)

	if err != nil {
		panic(err)
	}

	if w.Type == "io" {
		w.Keys = make([]string, ioLevel[w.Level].objectNumber)
		for i := 0; i < len(w.Keys); i++ {
			w.Keys[i] = fmt.Sprintf("in_%s_%d.bin", w.Name, i)
		}
	}

	runner = bencher.WithPayloadFunc(runner, w.Payload())

	if w.Operation != nil {
		delay := w.PhaseLength / time.Duration(2)
		runner = bencher.WithPhasePreRun(1, runner, func() error {
			go func() {
				time.Sleep(delay)
				log.Info("trigger operational change")
				err := w.Platform.Change(*w.Operation)
				if err != nil {
					log.Errorf("failed to apply OpTask %+v", err)
				}
			}()
			return nil
		})
	}

	return runner
}

func (w *PerformanceWorkload) GenerateIObjects() {
	//connect
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("auto"),
		Endpoint:         aws.String(w.Endpoint),
		Credentials:      credentials.NewStaticCredentials(w.AccessKeyID, w.AccessKeySecret, ""),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}))

	s3Client := s3.New(sess)

	//create a test bucket
	_, err := s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: &w.Bucket,
	})

	if err != nil {
		//Mhh
	}

	task := ioLevel[w.Level]
	//create #keys files
	for i := 0; i < len(w.Keys); i++ {
		key := w.Keys[i]
		body, size := generateRandomFile(task.objectSize)
		w.Keys[i] = key

		contentType := "application/octet-stream"
		_, err := s3Client.PutObject(&s3.PutObjectInput{
			Body:          body,
			Bucket:        &w.Bucket,
			ContentLength: &size,
			ContentType:   &contentType,
			Key:           &key,
		})

		if err != nil {
			//Panic
		}

	}
}

func (w PerformanceWorkload) Payload() bencher.PayloadFunc {
	if w.Level < 0 && w.Level > 7 {
		panic("workload complexity level unknown")
	}

	switch w.Type {
	case "idle":
		sleep := idleLevel[w.Level]
		data, err := json.Marshal(Job{Idle: &sleep})
		if err != nil {
			panic(err)
		}
		return func(invoker bencher.Invoker) []byte {
			return data
		}
	case "memory":
		task := memoryLevel[w.Level]
		data, err := json.Marshal(Job{Memory: &task})
		if err != nil {
			panic(err)
		}
		return func(invoker bencher.Invoker) []byte {
			return data
		}
	case "io":
		ioTemplate := ioLevel[w.Level]
		io := IOTask{
			Iteration:       ioTemplate.Iteration,
			ReadWrite:       ioTemplate.ReadWrite,
			ChunkSize:       ioTemplate.ChunkSize,
			Bucket:          w.Bucket,
			Keys:            w.Keys,
			Endpoint:        w.Endpoint,
			AccessKeyID:     w.AccessKeyID,
			AccessKeySecret: w.AccessKeySecret,
			Args: map[string]string{
				"DisableSSL":  strconv.FormatBool(w.DisableSSL),
				"S3PathStyle": strconv.FormatBool(w.S3PathStyle),
				"region":      w.S3Region,
			},
		}
		data, err := json.Marshal(Job{IO: &io})
		if err != nil {
			panic(err)
		}
		return func(invoker bencher.Invoker) []byte {
			return data
		}
	case "prime":
		level := primeLevel[w.Level]
		return func(invoker bencher.Invoker) []byte {
			primeCandidate := uint32(rand.Int31n(level) + rand.Int31n(level) - 1)
			data, err := json.Marshal(Job{Prime: &primeCandidate})
			if err != nil {
				log.Errorf("failed to generate prime payload %+v", err)
			}
			return data
		}
	}

	panic(fmt.Sprintf("workload of unknown type %s", w.Type))
}

type Job struct {
	Prime  *uint32     `json:"prime,omitempty"`
	Memory *MemoryTask `json:"memory,omitempty"`
	IO     *IOTask     `json:"io,omitempty"`
	Idle   *int        `json:"idle,omitempty"`
}

type MemoryTask struct {
	OperatorSize   uint32 `json:"operator_size,omitempty"`
	Iteration      uint32 `json:"itterations,omitempty"`
	RecursionDepth uint32 `json:"recursion_depth,omitempty"`
}

type IOTask struct {
	Iteration int     `json:"itteration,omitempty"`
	ReadWrite float32 `json:"rw,omitempty"`
	ChunkSize int64   `json:"size,omitempty"`

	Bucket string   `json:"bucket,omitempty"`
	Keys   []string `json:"keys,omitempty"`

	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"key_id,omitempty"`
	AccessKeySecret string `json:"key,omitempty"`

	objects map[string]int64

	Args map[string]string `json:"-"`

	objectSize   int64
	objectNumber int
}

func generateRandomFile(fileSize int64) (io.ReadSeeker, int64) {
	size := fileSize
	data := make([]byte, size)
	rand.Read(data)
	return bytes.NewReader(data), size
}
