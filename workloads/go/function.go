package bencher

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/faas-facts/fact/fact"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	factc "github.com/faas-facts/fact-go-client"
)

var contentType = "application/octet-stream"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Handle(client *factc.FactClient, job Job, context interface{}) fact.Trace {
	client.Start(context, nil)
	if job.Idle != nil {
		dur := time.Duration(*job.Idle) * time.Second
		Idle(&dur)
		client.Update(context, nil, map[string]string{
			"job": "idle",
		})
	} else if job.Prime != nil {
		Prime(*job.Prime)
		client.Update(context, nil, map[string]string{
			"job": "prime",
		})
	} else if job.Memory != nil {
		Memory(job.Memory)
		client.Update(context, nil, map[string]string{
			"job": "memory",
		})
	} else if job.IO != nil {
		r, w, e, err := IO(job.IO)
		client.Update(context, nil, map[string]string{
			"job": "io",
		})
		if err != nil {
			msg := err.Error()
			client.Update(context, &msg, map[string]string{})
		} else {
			client.Update(context, nil, map[string]string{
				"read":   strconv.FormatInt(r, 10),
				"writen": strconv.FormatInt(w, 10),
				"errors": strconv.FormatInt(int64(e), 10),
			})
		}
	}

	return client.Done(context, nil)
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

func ParallelMemory(task *MemoryTask, processes uint) {
	if task == nil {
		return
	}

	left := generateOperatorArray(task.OperatorSize)
	right := generateOperatorArray(task.OperatorSize)

	for p := uint(0); p < processes; p++ {
		go func() {
			for i := uint32(0); i < task.Iteration/uint32(processes); i++ {
				compute(0, task.RecursionDepth, left, right)
			}
		}()
	}

}

func Memory(task *MemoryTask) {
	if task == nil {
		return
	}

	left := generateOperatorArray(task.OperatorSize)
	right := generateOperatorArray(task.OperatorSize)

	for i := uint32(0); i < task.Iteration; i++ {
		compute(0, task.RecursionDepth, left, right)

		if i%100 == 0 {
			//increase allocations per 100 itterations
			tmp := make([]float64, len(right))
			copy(tmp, right)
			copy(right, left)
			copy(left, tmp)
		}
	}

}

func generateOperatorArray(size uint32) []float64 {
	data := make([]float64, size)
	for i := 0; i < len(data); i++ {
		data[i] = rand.Float64() * float64(rand.Int63())
	}
	return data
}

func compute(i, anchor uint32, left, right []float64) {
	if i < anchor {
		compute(i+1, anchor, left, right)
	} else {
		randomOpt(left, right)
	}
}

func randomOpt(left, right []float64) {
	a := left[rand.Intn(len(left))]
	b := right[rand.Intn(len(right))]

	var c float64
	if rand.Float32() < 0.5 {
		c = a * b
	} else {
		if b > 0 {
			c = a / b
		} else {
			c = a * b
		}
	}

	if rand.Float32() < 0.5 {
		left[rand.Intn(len(left))] = c
	} else {
		right[rand.Intn(len(right))] = c
	}
}

func Idle(delay *time.Duration) {
	if delay != nil {
		time.Sleep(*delay)
	}
}

//copy form http://www.rosettacode.org/wiki/Miller%E2%80%93Rabin_primality_test#Go
//Miller Rabin Algorithm
func Prime(n uint32) bool {
	// bases of 2, 7, 61 are sufficient to cover 2^32
	switch n {
	case 0, 1:
		return false
	case 2, 7, 61:
		return true
	}
	// compute s, d where 2^s * d = n-1
	nm1 := n - 1
	d := nm1
	s := 0
	for d&1 == 0 {
		d >>= 1
		s++
	}
	n64 := uint64(n)
	for _, a := range []uint32{2, 7, 61} {
		// compute x := a^d % n
		x := uint64(1)
		p := uint64(a)
		for dr := d; dr > 0; dr >>= 1 {
			if dr&1 != 0 {
				x = x * p % n64
			}
			p = p * p % n64
		}
		if x == 1 || uint32(x) == nm1 {
			continue
		}
		for r := 1; ; r++ {
			if r >= s {
				return false
			}
			x = x * x % n64
			if x == 1 {
				return false
			}
			if uint32(x) == nm1 {
				break
			}
		}
	}
	return true
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
}

func getStringFlag(key, defaultValue string, args map[string]string) string {
	if val, ok := args[key]; ok {
		return val
	} else {
		return defaultValue
	}
}

func getBoolFlag(key string, args map[string]string) bool {
	if val, ok := args[key]; ok {
		return strings.TrimSpace(strings.ToLower(val)) == "true"
	} else {
		return false
	}
}

func IO(task *IOTask) (int64, int64, int, error) {
	if task == nil {
		return -1, -1, 1, nil
	}

	//1 setup a connection
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(getStringFlag("region", "auto", task.Args)),
		Endpoint:         aws.String(task.Endpoint),
		Credentials:      credentials.NewStaticCredentials(task.AccessKeyID, task.AccessKeySecret, ""),
		DisableSSL:       aws.Bool(getBoolFlag("DisableSSL", task.Args)),
		S3ForcePathStyle: aws.Bool(getBoolFlag("S3PathStyle", task.Args)),
	}))

	client := s3.New(sess)

	task.objects = make(map[string]int64)
	for _, key := range task.Keys {
		head := s3.HeadObjectInput{
			Bucket: &task.Bucket,
			Key:    &key,
		}
		object, err := client.HeadObject(&head)
		if err != nil {
			log.Errorf("head %s error %f", key, err)
			return -1, -1, 1, err
		}
		task.objects[key] = *object.ContentLength
	}

	reads := int64(0)
	writes := int64(0)
	errors := 0
	for i := 0; i < task.Iteration; i++ {
		if rand.Float32() < task.ReadWrite {
			key := task.Keys[rand.Intn(len(task.Keys))]
			start := rand.Int63n(task.objects[key] - task.ChunkSize)
			rangeString := fmt.Sprintf("%d-%d", start, start+task.ChunkSize)
			get := s3.GetObjectInput{
				Bucket: &task.Bucket,
				Key:    &key,
				Range:  &rangeString,
			}
			object, err := client.GetObject(&get)
			if err != nil {
				errors++
				log.Errorf("read error %f", err)
			} else {
				//actually consume the body...
				written, err := io.Copy(ioutil.Discard, object.Body)
				if err != nil {
					errors++
					log.Errorf("read error %f", err)
				}
				reads += written
			}
			//read
		} else {
			key := fmt.Sprintf("generated_%d.bin", i)
			body := randomBytes(task.ChunkSize)
			put := s3.PutObjectInput{
				Body:          body,
				Bucket:        &task.Bucket,
				ContentLength: &task.ChunkSize,
				ContentType:   &contentType,
				Key:           &key,
			}
			_, err := client.PutObject(&put)
			if err != nil {
				errors++
				log.Errorf("write error %f", err)
			} else {
				writes += task.ChunkSize
			}
		}
	}
	return reads, writes, errors, nil
}

func randomBytes(chucksize int64) io.ReadSeeker {
	data := make([]byte, chucksize)
	n, err := rand.Read(data)
	if err != nil || int64(n) < chucksize {
		log.Print("failed to generate random bytes")
	}
	return bytes.NewReader(data)
}
