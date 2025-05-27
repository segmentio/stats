package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/segmentio/stats/v5/util/objconv"
	_ "github.com/segmentio/stats/v5/util/objconv/cbor"
	_ "github.com/segmentio/stats/v5/util/objconv/json"
	_ "github.com/segmentio/stats/v5/util/objconv/msgpack"
	_ "github.com/segmentio/stats/v5/util/objconv/resp"
	_ "github.com/segmentio/stats/v5/util/objconv/yaml"
)

// document is used to preserve the order of keys in maps.
type document []item

type item struct {
	K interface{}
	V interface{}
}

func (doc document) EncodeValue(e objconv.Encoder) error {
	i := 0
	return e.EncodeMap(len(doc), func(k objconv.Encoder, v objconv.Encoder) (err error) {
		if err = k.Encode(doc[i].K); err != nil {
			return
		}
		if err = v.Encode(doc[i].V); err != nil {
			return
		}
		i++
		return
	})
}

func (doc *document) DecodeValue(d objconv.Decoder) error {
	return d.DecodeMap(func(k objconv.Decoder, v objconv.Decoder) (err error) {
		var item item
		if err = k.Decode(&item.K); err != nil {
			return
		}
		if err = v.Decode(&item.V); err != nil {
			return
		}
		*doc = append(*doc, item)
		return
	})
}

func main() {
	var r = bufio.NewReader(os.Stdin)
	var w = bufio.NewWriter(os.Stdout)
	var input string
	var output string
	var list bool
	var pretty bool

	flag.StringVar(&input, "i", "json", "The format of the input stream")
	flag.StringVar(&output, "o", "json", "The format of the output stream")
	flag.BoolVar(&list, "l", false, "Prints a list of all the formats available")
	flag.BoolVar(&pretty, "p", false, "Prints in pretty format when available")
	flag.Parse()

	if list {
		codecs(os.Stdout)
		return
	}

	if err := conv(w, output, r, input, pretty); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	w.Flush()
}

func codecs(w io.Writer) {
	var names []string
	for name := range objconv.Codecs() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "- %s\n", name)
	}
	return
}

func conv(w io.Writer, output string, r io.Reader, input string, pretty bool) (err error) {
	var ic objconv.Codec
	var oc objconv.Codec
	var ok bool

	if ic, ok = objconv.Lookup(input); !ok {
		err = fmt.Errorf("unknown input format: %s", input)
		return
	}

	if oc, ok = objconv.Lookup(output); !ok {
		err = fmt.Errorf("unknown output format: %s", output)
		return
	}

	var d = objconv.NewStreamDecoder(ic.NewParser(r))
	var e *objconv.StreamEncoder
	var v interface{}
	var m = oc.NewEmitter(w)

	if pretty {
		if p, ok := m.(objconv.PrettyEmitter); ok {
			m = p.PrettyEmitter()
		}
	}

	if e, err = d.Encoder(m); err != nil {
		if err == io.EOF { // empty input
			err = nil
		}
		return
	}

	// Overwrite the type used for decoding maps so we can preserve the order
	// of the keys.
	d.MapType = reflect.TypeOf(document(nil))

	for d.Decode(&v) == nil {
		if err = e.Encode(v); err != nil {
			return
		}
		v = nil
	}

	if err = e.Close(); err != nil {
		return
	}

	// Not ideal but does the job, if the output is JSON we add a newline
	// character at the end to make it easier to read in terminals.
	if strings.Contains(output, "json") {
		fmt.Fprintln(w)
	}

	err = d.Err()
	return
}
