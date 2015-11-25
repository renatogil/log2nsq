# README #

Logger 2 NSQ for Golang projects

### How does it work? ###

Simple code example below:

```go
package main

import (
    l2n "sonaesr.com/log2nsq"
)

func main() {
    appname := "sample.logger"
    logger2Nsq := l2n.NewLog2Nsq(&l2n.Options{
		AppName: appname,
	})

    l2n.Printf("Starting application '%s'", appname)

    logger2Nsq.Close()
}
```

### Documentation ###

```
PACKAGE DOCUMENTATION

package log2nsq
    import "sonaesr.com/log2nsq"


FUNCTIONS

func Errorf(line string, params ...interface{})
    Errorf outputs data with the grade 'error'.

func Printf(line string, params ...interface{})
    Printf outputs data with the grade 'normal'.

func Println(line string)
    Println is the same as Printf, but without receiving any extra
    arguments.

func Tracef(line string, params ...interface{})
    Tracef outputs data with the grade 'debug'.

TYPES

type Log2Nsq struct {
    // contains filtered or unexported fields
}
    Log2Nsq main struc, representing the logger.

func NewLog2Nsq(opts *Options) *Log2Nsq
    NewLog2Nsq creates a new instance of the logger. It is a mandatory step
    before starting to log any data.

func (l2n *Log2Nsq) Close()
    Close closes all NSQ related comms. Defer it to the end of your
    application.

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
    Options for the Log2Nsq constructor.
```