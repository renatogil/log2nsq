package log2nsq

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bitly/go-nsq"
	"github.com/pborman/uuid"
)

// Options for the Log2Nsq constructor.
type Options struct {
	// Application name
	AppName string

	/* NSQ Address
	 * If ommited, a default one will be used
	 */
	Addr string

	// Additional tags to be included in the final JSON message sent to NSQ
	ExtraTags map[string]string
}

// Log2Nsq main struc, representing the logger.
type Log2Nsq struct {
	core     map[string]string
	producer *nsq.Producer
}

const topic string = "log.raw#ephemeral"

var l2n *Log2Nsq
var severity = [3]string{"debug", "info", "error"}

var connected bool
var errorQueue [][3]interface{}

func (l2n *Log2Nsq) buildFinalJSON(line string) map[string]map[string]string {
	json := make(map[string]string)
	for k, v := range l2n.core {
		json[k] = v
	}

	json["uuid"] = uuid.New()
	json["msg"] = line
	json["timestamp"] = time.Now().Format(time.RFC3339)

	data := make(map[string]map[string]string)
	data["data"] = json

	return data
}

func (l2n *Log2Nsq) publish(comboMap map[string]map[string]string) {
	if marshalled, err := json.Marshal(comboMap); err == nil {
		l2n.producer.Publish(topic, marshalled)
	} else {
		log.Printf("[log2nsq] Error publishing: %s\n", err.Error())
	}
}

// Close closes all NSQ related comms. Defer it to the end of your application.
func (l2n *Log2Nsq) Close() {
	l2n.producer.Stop()
}

func getHostname() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // Interface down
		}
		if iface.Flags&net.FlagLoopback == 0 {
			continue // Loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // Not an IPv4 address
			}

			return ip.String(), nil
		}
	}

	return "", errors.New("No IP address found")
}

func newLogger(opts *Options) *Log2Nsq {
	if len(opts.AppName) == 0 {
		log.Fatalln("[log2nsq] No application name defined")
	}

	if len(opts.Addr) == 0 {
		opts.Addr = "172.22.34.183:4150"
		log.Printf("[log2nsq] NSQ address not defined, using default %s\n", opts.Addr)
	}

	var hostname string
	var err error
	if hostname, err = getHostname(); err != nil {
		log.Printf("[log2nsq] Couldn't find a valid hostname: %s\n", err.Error())
		log.Println("[log2nsq] Using localhost to avoid stopping the application here")
		hostname = "127.0.0.1"
	}

	cfg := nsq.NewConfig()
	cfg.UserAgent = fmt.Sprintf("%s go-nsq/%s", opts.AppName, nsq.VERSION)

	producer, err := nsq.NewProducer(opts.Addr, cfg)
	if err != nil {
		log.Fatalf("[log2nsq] Failed to create NSQ producer @ %s\n", opts.Addr)
	}

	core := make(map[string]string)
	core["hostname"] = hostname
	core["application"] = opts.AppName

	for k, v := range opts.ExtraTags {
		core[k] = v
	}

	return &Log2Nsq{
		core:     core,
		producer: producer,
	}
}

func addToQueue(line string, sev string, params ...interface{}) {
	if !connected && len(errorQueue) == 0 {
		log.Println("[log2nsq] Queuing messages...")
	}

	log.Printf(line, params...)

	queueItem := [3]interface{}{line, sev, params}
	errorQueue = append(errorQueue, queueItem)
}

func emptyQueue() {
	for _, queueItem := range errorQueue {
		dumpToNsq(queueItem[0].(string), queueItem[1].(string), queueItem[2].([]interface{})...)
	}

	errorQueue = errorQueue[:0]
}

func dumpToNsq(line string, sev string, params ...interface{}) {
	if l2n == nil {
		addToQueue(line, sev, params...)
		return
	}

	if !connected {
		connected = true
		log.Println("[log2nsq] Dumping to NSQ")
		emptyQueue()
	}

	var comboMap map[string]map[string]string
	if params == nil {
		comboMap = l2n.buildFinalJSON(line)
	} else if params[0] == nil {
		comboMap = l2n.buildFinalJSON(line)
	} else {
		comboMap = l2n.buildFinalJSON(fmt.Sprintf(line, params...))
	}

	comboMap["data"]["severity"] = sev
	l2n.publish(comboMap)
}

// Tracef outputs data with the grade 'debug'.
func Tracef(line string, params ...interface{}) {
	dumpToNsq(line, severity[0], params...)
}

// Printf outputs data with the grade 'normal'.
func Printf(line string, params ...interface{}) {
	dumpToNsq(line, severity[1], params...)
}

// Println is the same as Printf, but without receiving any extra arguments.
func Println(line string) {
	dumpToNsq(line, severity[1], nil)
}

// Errorf outputs data with the grade 'error'.
func Errorf(line string, params ...interface{}) {
	dumpToNsq(line, severity[2], params...)
}

// NewLog2Nsq creates a new instance of the logger. It is a mandatory step before starting to log any data.
func NewLog2Nsq(opts *Options) *Log2Nsq {
	l2n = newLogger(opts)
	return l2n
}

func init() {
	connected = false
}
