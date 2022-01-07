package logrusrotate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const logPattern = "logrusrotate*.log"

func createTempfile() *os.File {
	f, err := os.CreateTemp(os.TempDir(), logPattern)
	if err != nil {
		panic(fmt.Sprintf("tempfile error: %s", err))
	}
	return f
}

func logFiles(basename string) []string {
	matches, err := filepath.Glob(basename + "*")
	if err != nil {
		panic(err)
	}
	return matches
}

func slurpFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read: %s", err))
	}
	return string(data)
}

func numLinks(path string) uint64 {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		panic(fmt.Sprintf("stat: %s", err))
	}
	return stat.Nlink
}

func setup() (basefile *os.File, log *logrus.Logger, remover func()) {
	basefile = createTempfile()
	log = logrus.New()
	remover = func() {
		for _, name := range logFiles(basefile.Name()) {
			os.Remove(name)
		}
	}
	return
}

func expectLinks(t *testing.T, nfiles int, basename string) {
	files := logFiles(basename)
	if len(files) != nfiles {
		t.Errorf("unexpected number of logs: %d, expected: %d", len(files), nfiles)
	}

	// fmt.Printf("logs: %v\n", files)

	for i, ff := range files {
		nlinks := numLinks(ff)
		expectedLinks := uint64(1)
		if i == 0 || i == len(files)-1 {
			expectedLinks = 2
		}
		if nlinks != expectedLinks {
			t.Errorf("unexpected number of links for %s, got: %d, expected: %d", ff, nlinks, expectedLinks)
		}
	}
}

func testWithFormat(t *testing.T, format string) {
	f, log, remover := setup()
	defer remover()

	lr, err := New(log, f.Name(), 5*Second, 0, format)
	if err != nil {
		t.Fatalf("error creating logrusrotate instance %s", err)
	}

	expectLinks(t, 2, f.Name())

	log.Info("line 1")
	log.Info("line 2")

	contents := slurpFile(f.Name())
	if !strings.Contains(contents, "line 1") || !strings.Contains(contents, "line 2") {
		t.Error("unexpected log contents")
	}

	fmt.Println("sleeping for 1 second...")
	time.Sleep(time.Second)
	err = lr.Rotate()
	if err != nil {
		t.Fatalf("rotation error %s", err)
	}

	expectLinks(t, 3, f.Name())

	fmt.Println("sleeping till next rotation kicks in...")
	time.Sleep(6 * time.Second)

	expectLinks(t, 4, f.Name())
}

func TestTimedFormat(t *testing.T) {
	testWithFormat(t, time.RFC3339)
}

func TestSequentialFormat(t *testing.T) {
	testWithFormat(t, "8")
}
