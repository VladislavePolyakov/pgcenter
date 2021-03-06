// Stuff related to Logtail AUXTYPE.

package top

import (
	"io"
	"os"
)

// Describes Postgres log and provides some metadata
type Logfile struct {
	Path       string   // Absolute path to logfile
	File       *os.File // Pointer to opened logfile
	Size       int64    // Size of the logfile (read file's content only when size grows)
	Bufsize    int      // Buffer to read in. It's unnecessary to read more than buffer size
	LinesLimit int      // Number of lines needed to show
}

var (
	pgLog Logfile
)

// Open logfile specified in Path variable.
func (l *Logfile) Open() error {
	var err error
	l.File, err = os.Open(l.Path)
	if err != nil {
		return err
	}

	return nil
}

// Close logfile.
func (l *Logfile) Close() error {
	return l.File.Close()
}

// Reopen logfile in case of rotate.
func (l *Logfile) ReOpen() error {
	var err error

	if err = l.Close(); err != nil {
		return err
	}

	if l.Path, err = readLogPath(); err != nil {
		return err
	}

	if err = l.Open(); err != nil {
		return err
	}

	return nil
}


// Read logfile until required number of newlines aren't collected
func (l *Logfile) Read() ([]byte, error) {
	var offset int64 = -1 // offset used for per-byte backward reading of the logfile
	var position int64    // position within the logfile from which reading starts
	var startpos int64    // final position from which reading of required amount of lines will start
	var newlines int      // newlines counter

	// Start reading from the end of file
	position, err := l.File.Seek(offset, 2)
	if err != nil {
		return nil, err
	}

	for i := 0; i < l.Bufsize; i++ {
		// The beginning of the file is reached, stop the reading
		if position < 0 {
			startpos = 0
			break
		}

		// Read 1 byte and check - is it a newline symbol? If symbol is a newline, remember this position - when number
		// of required newlines is reached, will start reading of logfile from this position to the buffer.
		c := make([]byte, 1)
		_, err := l.File.ReadAt(c, position)
		if err != nil {
			return nil, err
		}
		if string(c) == "\n" {
			newlines++
			startpos = position + 1 // +1 here, means that reading will start from symbol which is next after newline
		}

		// Stop reading when required number of newlines is reached
		if newlines > l.LinesLimit {
			break
		}

		offset--   // move 1 byte back
		position-- // shift position too
	}

	// Final reading of the logfile to buffer from calculated position
	buf := make([]byte, l.Bufsize)
	_, err = l.File.ReadAt(buf, startpos)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf, nil
}
