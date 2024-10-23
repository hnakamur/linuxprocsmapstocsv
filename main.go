package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"unicode/utf8"
)

// https://docs.kernel.org/filesystems/proc.html

type args struct {
	inputFilename  string
	outputFilename string
	Separator      string
}

type region struct {
	AddressStart []byte
	AddressEnd   []byte
	Perms        []byte
	Offset       []byte
	Dev          []byte
	Inode        []byte
	Pathname     []byte
}

type mapping struct {
	Region      *region
	FieldNames  []string
	FieldValues []string
}

var errBadFormat = errors.New("bad format")

const maxLineLength = 256

func main() {
	var args args
	flag.StringVar(&args.inputFilename, "i", "", "input filename to parse (in /proc/<pid>/smaps format)")
	flag.StringVar(&args.outputFilename, "o", "", "output CSV filename")
	flag.StringVar(&args.Separator, "sep", ",", "field separator")
	flag.Parse()

	if args.inputFilename == "" || args.outputFilename == "" {
		flag.Usage()
		log.Fatal("both flags -i and -o must be set")
	}
	if len(args.Separator) != 1 {
		log.Fatal("separator (-sep) must be one character")
	}

	if err := run(args); err != nil {
		log.Fatal(err)
	}
}

func run(args args) error {
	inputFile, err := os.Open(args.inputFilename)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(args.outputFilename)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	w := csv.NewWriter(outputFile)
	sep, _ := utf8.DecodeRuneInString(args.Separator)
	w.Comma = sep
	if err := convertSmapsToCsv(w, inputFile); err != nil {
		return err
	}
	return err
}

func convertSmapsToCsv(w *csv.Writer, r io.Reader) error {
	br := bufio.NewReaderSize(r, maxLineLength)
	var m mapping
	var firstLineFieldLabels []string
	regionIndex := -1
	var prevRegionLineNo int
	lineNo := 0
	for {
		line, err := readLine(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		lineNo++

		if isRegionLine(line) {
			regionIndex++

			if regionIndex > 0 {
				if regionIndex == 1 {
					if err := w.Write(m.toCSVHeader()); err != nil {
						return err
					}
					firstLineFieldLabels = m.FieldNames
				} else {
					if err := m.checkFieldNames(firstLineFieldLabels, prevRegionLineNo); err != nil {
						return err
					}
				}

				if err := w.Write(m.toCSVRecord()); err != nil {
					return err
				}
			}

			r, err := parseRegion(line)
			if err != nil {
				return err
			}
			m.clear()
			m.Region = r

			prevRegionLineNo = lineNo
		} else {
			name, value, err := parseField(line)
			if err != nil {
				return err
			}
			m.appendField(string(name), string(value))
		}
	}

	if err := m.checkFieldNames(firstLineFieldLabels, prevRegionLineNo); err != nil {
		return err
	}
	if err := w.Write(m.toCSVRecord()); err != nil {
		return err
	}
	w.Flush()

	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

const lf = '\n'

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes(lf)
	if err != nil {
		return nil, err
	}
	return bytes.TrimRight(line, "\n"), nil
}

func isRegionLine(line []byte) bool {
	// Region line contains ASCII space before colon
	// fcf0001000-fcf0002000 rw-p 00000000 00:00 0
	i := bytes.IndexByte(line, ':')
	if i == -1 {
		panic("unexpected line format, no colon found")
	}
	return bytes.IndexByte(line[:i], ' ') != -1
}

func parseRegion(line []byte) (*region, error) {
	addressStart, rest, ok := bytes.Cut(line, []byte{'-'})
	if !ok {
		return nil, errBadFormat
	}
	addressEnd, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, errBadFormat
	}
	perms, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, errBadFormat
	}
	offset, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, errBadFormat
	}
	dev, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, errBadFormat
	}
	inode, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, errBadFormat
	}
	pathname := bytes.TrimSpace(rest)
	return &region{
		AddressStart: addressStart,
		AddressEnd:   addressEnd,
		Perms:        perms,
		Offset:       offset,
		Dev:          dev,
		Inode:        inode,
		Pathname:     pathname,
	}, nil
}

func (m *mapping) clear() {
	m.Region = nil
	m.FieldNames = nil
	m.FieldValues = nil
}

func (m *mapping) appendField(name, value string) {
	m.FieldNames = append(m.FieldNames, name)
	m.FieldValues = append(m.FieldValues, value)
}

func (m *mapping) toCSVHeader() []string {
	return append([]string{
		"AddressStart",
		"AddressEnd",
		"Perms",
		"Offset",
		"Dev",
		"Inode",
		"Pathname",
	}, m.FieldNames...)
}

func (m *mapping) toCSVRecord() []string {
	return append([]string{
		string(m.Region.AddressStart),
		string(m.Region.AddressEnd),
		string(m.Region.Perms),
		string(m.Region.Offset),
		string(m.Region.Dev),
		string(m.Region.Inode),
		string(m.Region.Pathname),
	}, m.FieldValues...)
}

func (m *mapping) checkFieldNames(firstLineFieldNames []string, regionLineNo int) error {
	if !reflect.DeepEqual(m.FieldNames, firstLineFieldNames) {
		return fmt.Errorf("field names mismatch betweeen the first region and the region at line %d\n"+
			"fields in first region:%v\n"+
			"feilds in region at line %d:%v",
			regionLineNo, firstLineFieldNames,
			regionLineNo, m.FieldNames)
	}
	return nil
}

func parseField(line []byte) (name, value []byte, err error) {
	name, rest, ok := bytes.Cut(line, []byte{':'})
	if !ok {
		return nil, nil, errBadFormat
	}

	value = bytes.TrimLeft(rest, " ")
	if !bytes.Equal(name, []byte("VmFlags")) {
		value, _, _ = bytes.Cut(value, []byte{' '})
	}
	return name, value, nil
}
