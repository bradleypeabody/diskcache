package diskcache

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

var ErrNotFound = fmt.Errorf("Item not found")

// simple disk based cache
type DiskCache struct {
	Dir          string
	MaxBytes     int64
	MaxFiles     int64
	CleanupSleep time.Duration
	Shutdown     chan interface{}
}

// new disk cache with sensible defaults
func NewDiskCache() *DiskCache {
	return &DiskCache{
		Dir:          os.TempDir(),
		MaxBytes:     1 << 20, // 1mb
		MaxFiles:     256,
		CleanupSleep: 60 * time.Second,
	}
}

func (c *DiskCache) Start() error {

	if c.MaxBytes <= 0 {
		return fmt.Errorf("MaxBytes cannot be <= 0")
	}

	if c.MaxFiles <= 0 {
		return fmt.Errorf("MaxFiles cannot be <= 0")
	}

	if c.CleanupSleep <= 0 {
		return fmt.Errorf("CleanupSleep cannot be <= 0")
	}

	c.Shutdown = make(chan interface{}, 1)

	go func() {

		ticker := time.NewTicker(c.CleanupSleep)

		for {

			select {
			case <-ticker.C:
				err := c.cleanup()
				if err != nil {
					log.Printf("Error during cleanup: %v", err)
				}
			case <-c.Shutdown:
				log.Printf("Shutting down disk cache")
				return
			}

		}

	}()

	return nil
}

// Read file contents from cache, returns ErrNotFound if not there
func (c *DiskCache) Get(fname string) ([]byte, error) {

	p := filepath.Join(c.Dir, fname)

	// update timestamp
	now := time.Now()
	os.Chtimes(p, now, now)

	// read file contents
	b, err := ioutil.ReadFile(p)
	if err != nil {
		// FIXME: should do more to distinguish between file not found and other errors
		return nil, ErrNotFound
	}
	return b, nil
}

func (c *DiskCache) Set(fname string, val []byte) error {

	p := filepath.Join(c.Dir, fname)

	return ioutil.WriteFile(p, val, 0644)

}

func (c *DiskCache) cleanup() error {

	files := make(FDataList, 0, 256)

	err := filepath.Walk(c.Dir, func(p string, info os.FileInfo, err error) error {
		// skip anything that we failed to read info on
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, FData{
			Path:    path.Base(info.Name()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		log.Printf("Diskcache.cleanup() got error: %v", err)
		return nil
	}

	// sort by date
	sort.Sort(files)

	// trim down until size and file count limitations are met
	s := files.TotalSize()
	fcount := len(files)
	for i := 0; i < len(files); i++ {
		if s > c.MaxBytes || int64(fcount) > c.MaxFiles {
			s -= files[i].Size
			os.Remove(filepath.Join(c.Dir, files[i].Path))
			fcount--
		} else {
			break
		}
	}

	return nil
}

type FData struct {
	Path    string
	ModTime time.Time
	Size    int64
}
type FDataList []FData

func (f FDataList) Len() int {
	return len(f)
}

func (f FDataList) Less(i, j int) bool {
	return f[i].ModTime.Unix() < f[j].ModTime.Unix()
}

func (f FDataList) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FDataList) TotalSize() int64 {
	ret := int64(0)
	for _, f0 := range f {
		ret += f0.Size
	}
	return ret
}
