package filesystem

import (
	"errors"
	"fmt"
	"io"
	"sort"
)

var (
	ErrRange            = errors.New("range outside object")
	ErrSegmentTruncated = errors.New("segment file shorter than recorded size")
)

// Segment describes one file that makes up part of a logical object.
type Segment struct {
	FileID string
	Size   int64
	Start  int64 // cumulative offset of this segment within the object
}

// SegmentedReader presents an ordered list of files as one logical file.
// It holds no open descriptors: each segment is opened only while it is being
// copied, so it is cheap to create and safe for concurrent use.
type SegmentedReader struct {
	fm   *fileManager
	segs []Segment
	size int64
}

func (fm *fileManager) NewSegmentedReader(segs []Segment) *SegmentedReader {
	var size int64
	if n := len(segs); n > 0 {
		size = segs[n-1].Start + segs[n-1].Size
	}
	return &SegmentedReader{fm: fm, segs: segs, size: size}
}

func (sr *SegmentedReader) Size() int64 {
	return sr.size
}

func (sr *SegmentedReader) WriteRange(w io.Writer, start, length int64) error {
	if length == 0 {
		return nil
	}
	if start < 0 || length < 0 || start+length > sr.size {
		return ErrRange
	}

	// first segment whose end is beyond start
	i := sort.Search(len(sr.segs), func(i int) bool {
		return sr.segs[i].Start+sr.segs[i].Size > start
	})
	if i == len(sr.segs) {
		return ErrRange
	}
	off := start - sr.segs[i].Start

	for length > 0 && i < len(sr.segs) {
		seg := sr.segs[i]
		n := min(length, seg.Size-off)
		if err := sr.copySegment(w, seg, off, n); err != nil {
			return err
		}
		length -= n
		off = 0 // only the first segment is entered mid-way
		i++
	}
	return nil
}

func (sr *SegmentedReader) copySegment(w io.Writer, seg Segment, off, n int64) error {
	f, err := sr.fm.GetFile(seg.FileID)
	if err != nil {
		return fmt.Errorf("open segment %s: %w", seg.FileID, err)
	}
	defer f.Close()

	if off > 0 {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return fmt.Errorf("seek segment %s: %w", seg.FileID, err)
		}
	}
	if _, err := io.CopyN(w, f, n); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: %s", ErrSegmentTruncated, seg.FileID)
		}
		return err
	}
	return nil
}

func (sr *SegmentedReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= sr.size {
		return 0, io.EOF
	}
	i := sort.Search(len(sr.segs), func(i int) bool {
		return sr.segs[i].Start+sr.segs[i].Size > off
	})
	n := 0
	for n < len(p) && i < len(sr.segs) {
		seg := sr.segs[i]
		f, err := sr.fm.GetFile(seg.FileID)
		if err != nil {
			return n, err
		}
		inSeg := off + int64(n) - seg.Start
		want := min(int64(len(p)-n), seg.Size-inSeg)
		m, err := f.ReadAt(p[n:n+int(want)], inSeg)
		f.Close()
		n += m
		if err != nil && !errors.Is(err, io.EOF) {
			return n, err
		}
		if int64(m) < want {
			return n, ErrSegmentTruncated
		}
		i++
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

var _ io.ReaderAt = (*SegmentedReader)(nil)
