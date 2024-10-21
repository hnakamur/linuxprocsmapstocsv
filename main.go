package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"unicode/utf8"
)

// https://docs.kernel.org/filesystems/proc.html

type Args struct {
	inputFilename  string
	outputFilename string
	Separator      string
}

type Region struct {
	AddressStart []byte
	AddressEnd   []byte
	Perms        []byte
	Offset       []byte
	Dev          []byte
	Inode        []byte
	Pathname     []byte
}

type Mapping struct {
	Region          *Region
	Size            string
	KernelPageSize  string
	MMUPageSize     string
	Rss             string
	Pss             string
	Pss_Dirty       string
	Shared_Clean    string
	Shared_Dirty    string
	Private_Clean   string
	Private_Dirty   string
	Referenced      string
	Anonymous       string
	KSM             string
	LazyFree        string
	AnonHugePages   string
	ShmemPmdMapped  string
	FilePmdMapped   string
	Shared_Hugetlb  string
	Private_Hugetlb string
	Swap            string
	SwapPss         string
	Locked          string
	THPeligible     string
	ProtectionKey   string
	VmFlags         string
}

var ErrBadFormat = errors.New("bad format")

const maxLineLength = 256

func main() {
	var args Args
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

func run(args Args) error {
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
	if err := w.Write([]string{
		"AddressStart",
		"AddressEnd",
		"Perms",
		"Offset",
		"Dev",
		"Inode",
		"Pathname",
		"Size",
		"KernelPageSize",
		"MMUPageSize",
		"Rss",
		"Pss",
		"Pss_Dirty",
		"Shared_Clean",
		"Shared_Dirty",
		"Private_Clean",
		"Private_Dirty",
		"Referenced",
		"Anonymous",
		"KSM",
		"LazyFree",
		"AnonHugePages",
		"ShmemPmdMapped",
		"FilePmdMapped",
		"Shared_Hugetlb",
		"Private_Hugetlb",
		"Swap",
		"SwapPss",
		"Locked",
		"THPeligible",
		"ProtectionKey",
		"VmFlags",
	}); err != nil {
		return err
	}

	br := bufio.NewReaderSize(inputFile, maxLineLength)
	for {
		// log.Printf("readling line %d", i)
		m, err := readMapping(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := w.Write(m.toCSVRecord()); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	return nil
}

const LF = '\n'

func readMapping(r *bufio.Reader) (*Mapping, error) {
	line, err := r.ReadBytes(LF)
	if err != nil {
		return nil, err
	}
	// log.Printf("region line=%s", string(line))
	region, err := parseRegion(line)
	if err != nil {
		return nil, err
	}
	m, err := readFields(r)
	if err != nil {
		return nil, err
	}
	m.Region = region
	return m, nil
}

func parseRegion(line []byte) (*Region, error) {
	addressStart, rest, ok := bytes.Cut(line, []byte{'-'})
	if !ok {
		return nil, ErrBadFormat
	}
	addressEnd, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, ErrBadFormat
	}
	perms, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, ErrBadFormat
	}
	offset, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, ErrBadFormat
	}
	dev, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, ErrBadFormat
	}
	inode, rest, ok := bytes.Cut(rest, []byte{' '})
	if !ok {
		return nil, ErrBadFormat
	}
	pathname := bytes.TrimSpace(rest)
	return &Region{
		AddressStart: addressStart,
		AddressEnd:   addressEnd,
		Perms:        perms,
		Offset:       offset,
		Dev:          dev,
		Inode:        inode,
		Pathname:     pathname,
	}, nil
}

func readFields(r *bufio.Reader) (*Mapping, error) {
	var m Mapping
	if err := readField(r, "Size", &m.Size); err != nil {
		return nil, err
	}
	if err := readField(r, "KernelPageSize", &m.KernelPageSize); err != nil {
		return nil, err
	}
	if err := readField(r, "MMUPageSize", &m.MMUPageSize); err != nil {
		return nil, err
	}
	if err := readField(r, "Rss", &m.Rss); err != nil {
		return nil, err
	}
	if err := readField(r, "Pss", &m.Pss); err != nil {
		return nil, err
	}
	if err := readField(r, "Pss_Dirty", &m.Pss_Dirty); err != nil {
		return nil, err
	}
	if err := readField(r, "Shared_Clean", &m.Shared_Clean); err != nil {
		return nil, err
	}
	if err := readField(r, "Shared_Dirty", &m.Shared_Dirty); err != nil {
		return nil, err
	}
	if err := readField(r, "Private_Clean", &m.Private_Clean); err != nil {
		return nil, err
	}
	if err := readField(r, "Private_Dirty", &m.Private_Dirty); err != nil {
		return nil, err
	}
	if err := readField(r, "Referenced", &m.Referenced); err != nil {
		return nil, err
	}
	if err := readField(r, "Anonymous", &m.Anonymous); err != nil {
		return nil, err
	}
	if err := readField(r, "KSM", &m.KSM); err != nil {
		return nil, err
	}
	if err := readField(r, "LazyFree", &m.LazyFree); err != nil {
		return nil, err
	}
	if err := readField(r, "AnonHugePages", &m.AnonHugePages); err != nil {
		return nil, err
	}
	if err := readField(r, "ShmemPmdMapped", &m.ShmemPmdMapped); err != nil {
		return nil, err
	}
	if err := readField(r, "FilePmdMapped", &m.FilePmdMapped); err != nil {
		return nil, err
	}
	if err := readField(r, "Shared_Hugetlb", &m.Shared_Hugetlb); err != nil {
		return nil, err
	}
	if err := readField(r, "Private_Hugetlb", &m.Private_Hugetlb); err != nil {
		return nil, err
	}
	if err := readField(r, "Swap", &m.Swap); err != nil {
		return nil, err
	}
	if err := readField(r, "SwapPss", &m.SwapPss); err != nil {
		return nil, err
	}
	if err := readField(r, "Locked", &m.Locked); err != nil {
		return nil, err
	}
	if err := readField(r, "THPeligible", &m.THPeligible); err != nil {
		return nil, err
	}
	if err := readField(r, "ProtectionKey", &m.ProtectionKey); err != nil {
		return nil, err
	}
	if err := readField(r, "VmFlags", &m.VmFlags); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Mapping) toCSVRecord() []string {
	return []string{
		string(m.Region.AddressStart),
		string(m.Region.AddressEnd),
		string(m.Region.Perms),
		string(m.Region.Offset),
		string(m.Region.Dev),
		string(m.Region.Inode),
		string(m.Region.Pathname),
		m.Size,
		m.KernelPageSize,
		m.MMUPageSize,
		m.Rss,
		m.Pss,
		m.Pss_Dirty,
		m.Shared_Clean,
		m.Shared_Dirty,
		m.Private_Clean,
		m.Private_Dirty,
		m.Referenced,
		m.Anonymous,
		m.KSM,
		m.LazyFree,
		m.AnonHugePages,
		m.ShmemPmdMapped,
		m.FilePmdMapped,
		m.Shared_Hugetlb,
		m.Private_Hugetlb,
		m.Swap,
		m.SwapPss,
		m.Locked,
		m.THPeligible,
		m.ProtectionKey,
		m.VmFlags,
	}
}

func readField(r *bufio.Reader, name string, field *string) error {
	line, err := r.ReadBytes(LF)
	if err != nil {
		return err
	}
	line = bytes.TrimRight(line, string(LF))
	gotName, rest, ok := bytes.Cut(line, []byte{':'})
	if !ok || string(gotName) != name {
		return ErrBadFormat
	}

	rest = bytes.TrimLeft(rest, " ")
	var value []byte
	switch name {
	case "THPeligible", "ProtectionKey", "VmFlags":
		value = rest
	default:
		value, rest, ok = bytes.Cut(bytes.TrimLeft(rest, " "), []byte{' '})
		if !ok || !bytes.Equal(rest, []byte("kB")) {
			return ErrBadFormat
		}
	}
	*field = string(value)
	return nil
}

func (r *Region) writeCsv(w io.Writer) error {
	if _, err := w.Write(r.AddressStart); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.AddressEnd); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.Perms); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.Offset); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.Dev); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.Inode); err != nil {
		return err
	}
	if _, err := w.Write([]byte{','}); err != nil {
		return err
	}
	if _, err := w.Write(r.Pathname); err != nil {
		return err
	}
	return nil
}

func (r *Region) String() string {
	var b strings.Builder
	r.writeCsv(&b)
	return b.String()
}
