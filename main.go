// log2life
// by Brian C. Lane <bcl@brianlane.com>
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

/* commandline flags */
type cmdlineArgs struct {
	Logfile string  // Logfile to read
	Speed   float64 // Playback speed factor 1.0 == realtime
	Columns int     // Columns of Life world in cells
	Rows    int     // Rows of Life world in cells
	Port    int     // Port to connect to
	Host    string  // Host IP to connect to
}

/* commandline defaults */
var cfg = cmdlineArgs{
	Logfile: "",
	Speed:   1.0,
	Columns: 100,
	Rows:    100,
	Port:    3051,
	Host:    "127.0.0.1",
}

/* parseArgs handles parsing the cmdline args and setting values in the global cfg struct */
func init() {
	flag.Float64Var(&cfg.Speed, "speed", cfg.Speed, "Playback speed. 1.0 is realtime")
	flag.IntVar(&cfg.Columns, "columns", cfg.Columns, "Width of Life world in cells")
	flag.IntVar(&cfg.Rows, "rows", cfg.Rows, "Height of Life world in cells")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "Port to listen to")
	flag.StringVar(&cfg.Host, "host", cfg.Host, "Host IP to bind to")

	// first non flag argument is the logfile name
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [options] logfile:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
}

func main() {
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	filename := flag.Arg(0)

	var f *os.File
	var err error
	if filename == "-" {
		f = os.Stdin

		// When feeding log lines we don't want to delay
		cfg.Speed = 0
		fmt.Printf("Playback of <stdin> to %s:%d in realtime\n", cfg.Host, cfg.Port)
	} else {

		// Read logfile line by line
		f, err = os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Playback of %s to %s:%d at %0.1fx speed\n", filename, cfg.Host, cfg.Port, cfg.Speed)
	}

	lastTime := time.Time{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pattern, timestamp, err := LineToPattern(scanner.Text(), cfg.Columns, cfg.Rows)
		if err != nil {
			log.Print(err)
			continue
		}

		// When reading from stdin speed is set to 0 for no delay. When replaying a
		// log file it will use the timestamps to replay it in realtime unless -speed is passed
		// with a different value.
		noTime := time.Time{}
		if lastTime != noTime {
			delay := time.Duration(float64(timestamp.Sub(lastTime).Microseconds())*1/cfg.Speed) * time.Microsecond
			fmt.Printf("delaying %s\n", delay)
			time.Sleep(delay)
		}
		lastTime = timestamp

		fmt.Printf("%s\n", strings.Join(pattern, "\n"))
		err = SendPattern(cfg.Host, cfg.Port, pattern)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
		}

	}
	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// LineToPattern converts a log line to a Life 1.05 pattern with position based on the client IP
func LineToPattern(line string, width, height int) ([]string, time.Time, error) {

	// Get the IP and convert to x, y coordinated, scaled by columns, rows and 0, 0 at the center
	fields := strings.SplitN(line, " ", 4)
	if fields[0] == "-" || strings.TrimSpace(fields[0]) == "" {
		return []string{}, time.Time{}, fmt.Errorf("No client IP address")
	}
	x, y := IPToXY(fields[0], width, height)

	// Get the timestamp (will eventually return this and use it for timing)
	// [20/Nov/2022:02:27:49 +0000]
	fields = strings.SplitN(fields[3], "]", 2)
	//	timestamp := fields[0][1:]
	timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", fields[0][1:])
	if err != nil {
		return []string{}, time.Time{}, err
	}

	// XOR the data into an 8x8 bitpattern
	// TODO Scramble this a bit more, all the log data is 7 bit
	var data [8]byte
	var idx int
	for _, b := range []byte(fields[1]) {
		// Skip quotes
		if b == byte('"') {
			continue
		}
		data[idx] = data[idx] ^ b
		idx = (idx + 1) % 8
	}

	// Convert the data to a Life 1.05 pattern
	return MakeLife105(x, y, data), timestamp, nil
}

// IPToXY convert an IPv4 dotted quad into an X, Y coordinate
func IPToXY(addr string, width, height int) (x, y int) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return 0, 0
	}

	// Only using IPv4 right now so 4 bytes from the ip which are at the end
	// because it converts it to a IPv6 encoded IPv4
	// Use the upper 16 bits as x and lower 16 as y, scaled to the life world size
	// and with 0,0 at the center
	x = int(float64(int(ip[12])<<8+int(ip[13]))/0xffff*float64(width)) - width/2
	y = int(float64(int(ip[14])<<8+int(ip[15]))/0xffff*float64(height)) - height/2

	return x, y
}

// SendPattern POSTs a pattern to the life server and returns any errors
func SendPattern(host string, port int, pattern []string) error {
	data := strings.NewReader(strings.Join(pattern, "\n"))
	resp, err := http.Post(fmt.Sprintf("http://%s:%d", host, port), "text/plain", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	return err
}

// MakeLife105 converts an array of 8 bytes into a life 1.05 pattern string
func MakeLife105(x, y int, data [8]byte) []string {
	var pattern []string

	pattern = append(pattern, "#Life 1.05")
	pattern = append(pattern, "#D log2life ouput")
	pattern = append(pattern, "#N")
	pattern = append(pattern, fmt.Sprintf("#P %d %d", x, y))

	for _, b := range data {
		var line string
		for i := 0; i < 8; i++ {
			if b&0x80 == 0x80 {
				line = line + "*"
			} else {
				line = line + "."
			}

			b = b << 1
		}
		pattern = append(pattern, line)
	}

	return pattern
}
